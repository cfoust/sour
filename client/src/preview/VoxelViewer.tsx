import React, { useRef, useEffect, useState, useCallback } from "react";
import { Box, Slider, SliderTrack, SliderFilledTrack, SliderThumb, Spinner, Center } from "@chakra-ui/react";
import { decodePreview } from "./decoder";
import { createScene } from "./scene";

type VoxelViewerProps = {
  svoxUrl: string;
  height?: number;
  showClipSlider?: boolean;
};

export default function VoxelViewer({
  svoxUrl,
  height = 420,
  showClipSlider = false,
}: VoxelViewerProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const sceneRef = useRef<ReturnType<typeof createScene> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [clipValue, setClipValue] = useState(100);

  useEffect(() => {
    if (!canvasRef.current || !containerRef.current) return;

    let disposed = false;

    async function load() {
      try {
        const resp = await fetch(svoxUrl);
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        const buf = await resp.arrayBuffer();
        if (disposed) return;

        const data = decodePreview(buf);
        if (disposed) return;

        const canvas = canvasRef.current!;
        const container = containerRef.current!;
        const width = container.clientWidth;
        const h = height;

        const s = createScene(canvas, data, width, h);
        sceneRef.current = s;
        setLoading(false);

        // Set initial clip from data
        if (data.clipY > 0 && data.clipY < data.gridSize) {
          const normalized = data.clipY / data.gridSize;
          setClipValue(Math.round(normalized * 100));
          s.setClipHeight(normalized);
        }
      } catch (e) {
        console.error("Failed to load preview:", e);
        if (!disposed) {
          setError(true);
          setLoading(false);
        }
      }
    }

    load();

    return () => {
      disposed = true;
      sceneRef.current?.dispose();
      sceneRef.current = null;
    };
  }, [svoxUrl, height]);

  const handleClipChange = useCallback((val: number) => {
    setClipValue(val);
    sceneRef.current?.setClipHeight(val / 100);
  }, []);

  if (error) return null; // fall back to parent's alternative

  return (
    <Box ref={containerRef} position="relative" w="100%" h={`${height}px`}>
      {loading && (
        <Center position="absolute" inset={0} zIndex={1}>
          <Spinner size="lg" color="white" />
        </Center>
      )}
      <canvas
        ref={canvasRef}
        style={{
          width: "100%",
          height: `${height}px`,
          display: "block",
          borderRadius: "inherit",
        }}
      />
      {showClipSlider && !loading && (
        <Box
          position="absolute"
          right="12px"
          top="50%"
          transform="translateY(-50%)"
          h="60%"
          zIndex={2}
        >
          <Slider
            aria-label="clip-height"
            orientation="vertical"
            min={10}
            max={100}
            value={clipValue}
            onChange={handleClipChange}
            h="100%"
          >
            <SliderTrack bg="whiteAlpha.300">
              <SliderFilledTrack bg="whiteAlpha.600" />
            </SliderTrack>
            <SliderThumb boxSize={4} />
          </Slider>
        </Box>
      )}
    </Box>
  );
}
