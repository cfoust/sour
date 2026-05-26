import * as React from 'react'
import styled from '@emotion/styled'
import { t } from './theme'
import { usePreviewStill } from '../preview/urls'

const Wrapper = styled.div`
  position: absolute;
  inset: 0;
`

const Img = styled.img`
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
`

const Missing = styled.div`
  position: absolute;
  inset: 0;
  background: ${t.surface2};
  display: flex;
  align-items: center;
  justify-content: center;
`

const MissingLabel = styled.span`
  font-family: ${t.fontMono};
  font-size: 11px;
  letter-spacing: 0.04em;
  color: ${t.mute};
`

const Stripes = styled.div`
  position: absolute;
  inset: 0;
  background-image: repeating-linear-gradient(
    45deg,
    rgba(255, 255, 255, 0.06) 0 1px,
    transparent 1px 7px
  );
  pointer-events: none;
`

type Props = {
  imageUrl?: string
  name: string
}

export default function MapThumb({ imageUrl, name }: Props) {
  const [failed, setFailed] = React.useState(false)
  const stillUrl = usePreviewStill(name)
  const src = stillUrl || imageUrl

  if (!src || failed) {
    return (
      <Missing>
        <Stripes />
        <MissingLabel>screenshot pending</MissingLabel>
      </Missing>
    )
  }

  return (
    <Wrapper>
      <Img
        src={src}
        alt={name}
        loading="lazy"
        onError={() => setFailed(true)}
      />
    </Wrapper>
  )
}
