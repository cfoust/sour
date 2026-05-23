export type PreviewData = {
  maxDepth: number;
  gridSize: number;
  worldSize: number;
  palette: Uint8Array; // flat RGB, length = numPalette * 3
  numVoxels: number;
  voxelX: Uint8Array;
  voxelY: Uint8Array;
  voxelZ: Uint8Array;
  voxelColor: Uint8Array;
  voxelFlags: Uint8Array;
  voxelAO: Uint8Array;
  numEntities: number;
  entityX: Uint8Array;
  entityY: Uint8Array;
  entityZ: Uint8Array;
  entityType: Uint8Array;
  skyTop: [number, number, number];
  skyHorizon: [number, number, number];
  ambient: [number, number, number];
  sunlight: [number, number, number];
};

export function decodePreview(buffer: ArrayBuffer): PreviewData {
  const view = new DataView(buffer);
  const bytes = new Uint8Array(buffer);

  // Validate magic
  if (
    bytes[0] !== 0x53 || // S
    bytes[1] !== 0x56 || // V
    bytes[2] !== 0x4f || // O
    bytes[3] !== 0x58    // X
  ) {
    throw new Error("Invalid SVOX magic bytes");
  }

  const version = bytes[4];
  if (version !== 1) {
    throw new Error(`Unsupported SVOX version: ${version}`);
  }

  const maxDepth = bytes[5];
  const gridSize = view.getUint16(6, true);
  const worldSize = view.getUint32(8, true);
  const numPalette = view.getUint16(12, true);
  const numVoxels = view.getUint32(14, true);
  const numEntities = view.getUint16(18, true);

  const skyTop: [number, number, number] = [bytes[20], bytes[21], bytes[22]];
  const skyHorizon: [number, number, number] = [bytes[23], bytes[24], bytes[25]];
  const ambient: [number, number, number] = [bytes[26], bytes[27], bytes[28]];
  const sunlight: [number, number, number] = [bytes[29], bytes[30], bytes[31]];

  let off = 32;

  // Palette
  const palette = new Uint8Array(numPalette * 3);
  palette.set(bytes.subarray(off, off + numPalette * 3));
  off += numPalette * 3;

  // Voxels (SoA for better cache usage in rendering)
  const voxelX = new Uint8Array(numVoxels);
  const voxelY = new Uint8Array(numVoxels);
  const voxelZ = new Uint8Array(numVoxels);
  const voxelColor = new Uint8Array(numVoxels);
  const voxelFlags = new Uint8Array(numVoxels);
  const voxelAO = new Uint8Array(numVoxels);

  for (let i = 0; i < numVoxels; i++) {
    voxelX[i] = bytes[off];
    voxelY[i] = bytes[off + 1];
    voxelZ[i] = bytes[off + 2];
    voxelColor[i] = bytes[off + 3];
    voxelFlags[i] = bytes[off + 4];
    voxelAO[i] = bytes[off + 5];
    off += 6;
  }

  // Entities
  const entityX = new Uint8Array(numEntities);
  const entityY = new Uint8Array(numEntities);
  const entityZ = new Uint8Array(numEntities);
  const entityType = new Uint8Array(numEntities);

  for (let i = 0; i < numEntities; i++) {
    entityX[i] = bytes[off];
    entityY[i] = bytes[off + 1];
    entityZ[i] = bytes[off + 2];
    entityType[i] = bytes[off + 3];
    off += 4;
  }

  return {
    maxDepth,
    gridSize,
    worldSize,
    palette,
    numVoxels,
    voxelX,
    voxelY,
    voxelZ,
    voxelColor,
    voxelFlags,
    voxelAO,
    numEntities,
    entityX,
    entityY,
    entityZ,
    entityType,
    skyTop,
    skyHorizon,
    ambient,
    sunlight,
  };
}
