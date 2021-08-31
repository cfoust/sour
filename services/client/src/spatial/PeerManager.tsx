// @flow strict

import type { ClientID } from '../types'

import * as R from 'ramda'

const MEDIA_CONSTRAINTS = {
  audio: true,
  video: false,
}

type UserContext = {
  startTime: number
  endTime?: number

  connection: RTCPeerConnection
  element: HTMLAudioElement

  // This might not have been established yet.
  panner: Maybe<PannerNode>
}

export enum SpatialMessageType {
  WebRTC,
  Join,
  Leave,
}

export enum WebRTCMessageType {
  Offer,
  Answer,
  Candidate,
}

export type WebRTCOffer = {
  type: WebRTCMessageType.Offer
  offer: RTCSessionDescriptionInit
}

export type WebRTCAnswer = {
  type: WebRTCMessageType.Answer
  answer: RTCSessionDescriptionInit
}

export type WebRTCCandidate = {
  type: WebRTCMessageType.Candidate
  candidate: RTCIceCandidateInit
}

export type WebRTCExchange = WebRTCOffer | WebRTCCandidate | WebRTCAnswer

export type WebRTCMessage = {
  type: SpatialMessageType.WebRTC
  // The source user
  from: ClientID
  // The destination user
  to: ClientID
  exchange: WebRTCExchange
}

export type JoinMessage = {
  type: SpatialMessageType.Join
  id: ClientID
}

export type LeaveMessage = {
  type: SpatialMessageType.Leave
  id: ClientID
}

export type SpatialMessage = WebRTCMessage | JoinMessage | LeaveMessage

type MessageSender = (message: SpatialMessage) => Maybe<Promise<void>>
export type SpeakingHandler = (speaking: boolean) => void

// The amount of times we check for speech, in Hz.
const SAMPLE_RATE = 20
const SAMPLE_TIME = 1000 / SAMPLE_RATE
// The size of the circular buffer for detecting sound.
const SPEECH_CIRCULAR_SIZE = 8
// The volume above which we consider the user to be speaking.
// TODO(cfoust): 08/28/20 make this configurable
const VOLUME_THRESHOLD = -60

const NEGATIVE_INFINITY = -1 * Infinity
const FFT_SIZE = 512

declare class RTCPeerNegotiationEvent extends Event {
  currentTarget: RTCPeerConnection
}

export default class PeerManager {
  id: ClientID

  users: { [id: number]: UserContext }

  speakingHandler: SpeakingHandler

  stream: Maybe<MediaStream>

  volumeTimer: Maybe<IntervalID>

  isSpeaking: boolean

  isMuted: boolean

  ws: Maybe<WebSocket>

  constructor(clientId: number, speakingHandler: SpeakingHandler) {
    this.id = clientId
    this.speakingHandler = speakingHandler
    this.users = {}
    this.isSpeaking = false
    this.isMuted = false
  }

  connect() {
    const ws = new WebSocket(`ws://${window.location.hostname}:28786`)

    ws.onmessage = ({ data }) => {
      if (typeof data !== 'string') return
      const parsed: SpatialMessage = JSON.parse(data)
      this.receiveMessage(parsed)
    }

    ws.onopen = () => {
      this.sendMessage({
        type: SpatialMessageType.Join,
        id: this.id,
      })
    }

    this.ws = ws
  }

  setMuted(status: boolean) {
    const { stream } = this
    if (stream == null) return
    stream.getAudioTracks()[0].enabled = !status
    this.isMuted = status
    this.speakingHandler(false)
  }

  getUser(target: ClientID): Maybe<UserContext> {
    return this.users[target]
  }

  sendMessage(message: SpatialMessage) {
    this.ws?.send(JSON.stringify(message))
  }

  receiveMessage(message: SpatialMessage) {
    if (message.type === SpatialMessageType.Join) {
      this.connectToPeer(message.id)
      return
    }

    if (message.type === SpatialMessageType.Leave) {
      this.cleanupPeer(message.id)
      return
    }

    const { exchange, from } = message

    if (exchange.type === WebRTCMessageType.Offer) {
      this.handleOffer(from, exchange.offer)
    } else if (exchange.type === WebRTCMessageType.Answer) {
      this.handleAnswer(from, exchange.answer)
    } else if (exchange.type === WebRTCMessageType.Candidate) {
      this.handleRemoteIceCandidate(from, exchange.candidate)
    }
  }

  async sendWebRTC(destination: ClientID, message: WebRTCExchange) {
    this.sendMessage({
      type: SpatialMessageType.WebRTC,
      from: this.id,
      to: destination,
      exchange: message,
    })
  }

  async createPeerConnection(target: ClientID) {
    const connection = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
    })

    connection.onicecandidate = this.handleICECandidate.bind(this, target)
    connection.oniceconnectionstatechange = this.handleICEStateChange.bind(
      this,
      target
    )
    connection.onicegatheringstatechange = this.handleICEGatheringStateChange.bind(
      this,
      target
    )
    connection.onsignalingstatechange = this.handleSignalingChange.bind(
      this,
      target
    )
    // @ts-ignore
    connection.onnegotiationneeded = this.handleNegotiationNeeded.bind(
      this,
      target
    )
    connection.ontrack = this.handleTrack.bind(this, target)

    const element = document.createElement('audio')
    if (document.body) {
      document.body.appendChild(element)
    }
    element.autoplay = true
    element.muted = true

    this.users[target] = {
      startTime: Date.now(),
      connection,
      element,
      panner: null,
    }

    return connection
  }

  async getStream(): Promise<MediaStream> {
    const { mediaDevices } = navigator
    if (mediaDevices == null) {
      throw new Error('No media devices')
    }

    const { stream: existingStream } = this
    if (existingStream != null) return existingStream

    const stream = await mediaDevices.getUserMedia(MEDIA_CONSTRAINTS)

    // Create a new pipeline so we can intercept and mute
    const context = new AudioContext()
    const source = context.createMediaStreamSource(stream)
    const analyser = context.createAnalyser()
    analyser.fftSize = FFT_SIZE
    analyser.smoothingTimeConstant = 0.1

    const dataArray = new Float32Array(FFT_SIZE)

    const circularBuffer: boolean[] = R.map(
      R.F,
      R.range(0, SPEECH_CIRCULAR_SIZE)
    )
    let bufferIndex: number = 0
    let speakingCount: number = 0
    let silentCount: number = 0

    this.volumeTimer = setInterval(() => {
      analyser.getFloatFrequencyData(dataArray)

      const hasSpeech = R.pipe(
        // Some junky values
        (v: Float32Array) => Array.from(v).slice(4),
        (list: number[]) =>
          R.reduce(
            (i, v) =>
              R.maxBy(
                (a: number) => {
                  if (a >= 0) return NEGATIVE_INFINITY
                  return a
                },
                i,
                v
              ),
            NEGATIVE_INFINITY,
            list
          ),
        // Whether the max value is above the threshold
        R.lt(VOLUME_THRESHOLD)
      )(dataArray)

      speakingCount = Math.max(0, speakingCount + (hasSpeech ? 1 : -1))
      silentCount = speakingCount == 0 ? silentCount + 1 : 0

      circularBuffer[bufferIndex] = hasSpeech
      bufferIndex = bufferIndex++ % SPEECH_CIRCULAR_SIZE

      if (this.isMuted) return

      const isSpeaking = silentCount < SPEECH_CIRCULAR_SIZE || speakingCount > 0

      // We want to indicate the signal changes here to the handler
      if (!this.isSpeaking && isSpeaking) {
        // Started speaking
        this.speakingHandler(true)
      } else if (this.isSpeaking && !isSpeaking) {
        // Stopped speaking
        this.speakingHandler(false)
      }

      this.isSpeaking = isSpeaking
    }, SAMPLE_TIME)

    const destination = context.createMediaStreamDestination()

    source.connect(analyser)
    analyser.connect(destination)

    this.stream = destination.stream

    return destination.stream
  }

  async addTracks(connection: RTCPeerConnection) {
    const stream = await this.getStream()
    stream
      .getAudioTracks()
      .forEach((track) => connection.addTrack(track, stream))
  }

  async handleOffer(target: ClientID, message: RTCSessionDescriptionInit) {
    const connection = await this.createPeerConnection(target)

    const description = new RTCSessionDescription(message)
    await connection.setRemoteDescription(description)

    await this.addTracks(connection)

    const answer = await connection.createAnswer()
    await connection.setLocalDescription(answer)

    const { localDescription } = connection

    if (localDescription == null) {
      throw new Error('localDescription was null')
    }

    await this.sendWebRTC(target, {
      type: WebRTCMessageType.Answer,
      answer: localDescription,
    })
  }

  async handleAnswer(target: ClientID, message: RTCSessionDescriptionInit) {
    const peer = this.getUser(target)
    if (peer == null) return

    const { connection } = peer
    const description = new RTCSessionDescription(message)
    await connection.setRemoteDescription(description)
  }

  async handleRemoteIceCandidate(
    target: ClientID,
    candidate: RTCIceCandidateInit
  ) {
    const peer = this.getUser(target)
    if (peer == null) return
    const { connection } = peer

    const rtcCandidate = new RTCIceCandidate(candidate)
    await connection.addIceCandidate(rtcCandidate)
  }

  async handleICECandidate(target: ClientID, event: RTCPeerConnectionIceEvent) {
    const { candidate } = event
    if (candidate == null) return

    await this.sendWebRTC(target, {
      type: WebRTCMessageType.Candidate,
      candidate,
    })
  }

  async handleICEStateChange(target: ClientID, event: Event) {}

  async handleICEGatheringStateChange(target: ClientID, event: Event) {}

  async handleSignalingChange(target: ClientID, event: Event) {}

  async handleNegotiationNeeded(
    target: ClientID,
    event: RTCPeerNegotiationEvent
  ) {
    const { currentTarget: connection } = event

    const offer = await connection.createOffer()
    await connection.setLocalDescription(offer)

    const { localDescription } = connection

    if (localDescription == null) {
      throw new Error('localDescription was null')
    }

    await this.sendWebRTC(target, {
      type: WebRTCMessageType.Offer,
      offer: localDescription,
    })
  }

  async handleTrack(target: ClientID, event: RTCTrackEvent) {
    const { streams } = event
    const [first] = streams

    const peer = this.getUser(target)

    if (peer == null) {
      throw new Error('Could not find peer for target')
    }

    const { element } = peer

    if (element.srcObject === first) return

    element.srcObject = first

    const context = new AudioContext()
    const source = context.createMediaStreamSource(first)

    const panner = context.createPanner()
    panner.panningModel = 'HRTF'

    this.users[target] = {
      ...peer,
      endTime: Date.now(),
      panner,
    }

    source.connect(panner)
    panner.connect(context.destination)
  }

  async connectToPeer(target: ClientID) {
    const connection = await this.createPeerConnection(target)
    await this.addTracks(connection)
  }

  cleanupPeer(target: ClientID) {
    const { [target]: peer, ...rest } = this.users
    if (peer == null) return
    const { connection, element } = peer

    connection.close()
    element.remove()

    this.users = rest
  }

  terminate() {
    this.sendMessage({
      type: SpatialMessageType.Leave,
      id: this.id,
    })

    R.map(this.cleanupPeer.bind(this), R.keys(this.users))

    const { volumeTimer } = this
    if (volumeTimer != null) clearInterval(volumeTimer)
  }
}
