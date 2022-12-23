#Service: {
	enabled: bool | *true
}

proxy: #Service

#Server: {
	// An alias replaces the ID of a server. For example, users could run
	// #join lobby (on desktop) or /join lobby (on the web)
	alias:  string
	preset: string
}

#ServerPreset: {
	name: string

	// If this is true, the user cannot create servers with this preset,
	// it's only for inheritance or matchmaking purposes.
	virtual: bool | *false

	// If a config refers to a file that exists, we will just use that to start the server
	// config: /sour/config/server-init.cfg
	// You can also specify a CubeScript configuration for the server inline.
	config: string

	// This refers to another preset. That preset's configuration will be
	// prepended to this preset's configuration. This can be recursive.
	inherit?: string

	// This means that if the user does not specify a config, we use this preset.
	default: bool | *false
}

#DuelType: {
	name:   string
	preset: string
	// How long the warmup lasts
	warmupSeconds: uint | *30
	// How long the main match (after warmup lasts)
	gameSeconds: uint | *180
	// How many frags a player has to win by to win
	// (otherwise the game goes into overtime)
	winThreshold: uint | *3
	// The length of each overtime session. If a player is still not
	// winning by winThreshold, overtime is repeated.
	overtimeSeconds: uint | *60
	// "all" = force respawn everyone when someone dies
	// "dead" = force respawn just the person who died when they die
	// "none" = don't respawn anyone
	forceRespawn: "all" | "dead" | "none" | *"all"
	pauseOnDeath: bool | *false
	default:      bool | *false
}

#MatchmakingSettings: {
	duel: [...#DuelType]
}

#Port: uint16

#ENetIngress: {
	// The UDP port to listen on
	port: #Port
	// The name of the server to join when a client connects
	target: string
	// Configure the serverinfo port
	serverInfo: {
		enabled: bool | *false
		// Whether to register with the master server
		master: bool | *false
		// If defined, use the cluster's server info instead of the target info.
		cluster?: {
			description: string
			map:         string
		}
	}
}

// Describes all of the ways desktop clients can join this cluster
#IngressSettings: {
	desktop: [...#ENetIngress]
	web: {
		// The TCP port the WebSocket service should listen on.
		port: #Port
	}
}

// Sour clusters host game servers.
#ClusterSettings: {
	#Service

	// Server presets are templates for starting new servers, typically by a user through #creategame.
	// A mapping from preset name -> preset settings.
	presets: [...#ServerPreset]

	// This is not the same thing as client.assets because the cluster has to
	// specify complete URLs (and can access services using their addresses
	// inside the container.)
	assets: [...string]

	// These are all of the game servers that will be started when the cluster starts.
	servers: [...#Server]
	matchmaking: #MatchmakingSettings
	ingress:     #IngressSettings

	// We set the Sauerbraten `serverdesc` according to this template.
	// #id is replaced with the server's identifier.
	serverDescription: string | *"Sour [#id]"
}
cluster: #ClusterSettings

#ClientSettings: {
	#Service

	// All client URLs can use these template variables:
	// #host: replaced with window.location.host
	// #origin: replaced with window.location.origin (basically #protocol + #host)
	// #protocol: replaced with window.location.protocol e.g. https:

	// These are the URLs for each of the asset sources.
	// Order matters; the client uses the first map it finds.
	// The client's asset sources may not be the same as the cluster's because
	// we might not know the hostname the user will be accessing Sour at ahead
	// of time. We can take advantage of the browser's automatic addition of the
	// hostname to bare absolute paths.
	assets: [...string] | *["#origin/assets/.index.json"]

	// The URLs for all of the game servers, for now we only support one.
	// ws: and wss: are inferred
	clusters: [...string] | *["#host/service/cluster/"]
	// ws: and wss: are inferred
	proxy: string | *"#host/service/proxy/"
}
client: #ClientSettings
