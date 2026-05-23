export type PreviewData = {
  maxDepth: number;
  gridSize: number;
  worldSize: number;
  palette: Uint8Array; // flat RGB, length = numPalette * 3
  numVoxels: number;
  voxelX: Uint16Array;
  voxelY: Uint16Array;
  voxelZ: Uint16Array;
  voxelColor: Uint8Array;
  voxelFlags: Uint8Array;
  voxelAO: Uint8Array;
  numEntities: number;
  entityX: Uint16Array;
  entityY: Uint16Array;
  entityZ: Uint16Array;
  entityType: Uint8Array;
  skyTop: [number, number, number];
  skyHorizon: [number, number, number];
  ambient: [number, number, number];
  sunlight: [number, number, number];
  focusX: number;
  focusY: number;
  focusZ: number;
  focusRadius: number;
  cameraYaw: number;   // degrees
  cameraPitch: number; // degrees
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
  if (version !== 3) {
    throw new Error(`Unsupported SVOX version: ${version} (expected 3)`);
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

  const focusX = view.getUint16(32, true);
  const focusY = view.getUint16(34, true);
  const focusZ = view.getUint16(36, true);
  const focusRadius = view.getUint16(38, true);
  const cameraYaw = view.getUint16(40, true) / 10;   // tenths → degrees
  const cameraPitch = view.getUint16(42, true) / 10;  // tenths → degrees

  let off = 44;

  // Palette
  const palette = new Uint8Array(numPalette * 3);
  palette.set(bytes.subarray(off, off + numPalette * 3));
  off += numPalette * 3;

  // Voxels: X(u16) Y(u16) Z(u16) PaletteIndex(u8) Flags(u8) AO(u8) = 9 bytes
  const voxelX = new Uint16Array(numVoxels);
  const voxelY = new Uint16Array(numVoxels);
  const voxelZ = new Uint16Array(numVoxels);
  const voxelColor = new Uint8Array(numVoxels);
  const voxelFlags = new Uint8Array(numVoxels);
  const voxelAO = new Uint8Array(numVoxels);

  for (let i = 0; i < numVoxels; i++) {
    voxelX[i] = view.getUint16(off, true);
    voxelY[i] = view.getUint16(off + 2, true);
    voxelZ[i] = view.getUint16(off + 4, true);
    voxelColor[i] = bytes[off + 6];
    voxelFlags[i] = bytes[off + 7];
    voxelAO[i] = bytes[off + 8];
    off += 9;
  }

  // Entities: X(u16) Y(u16) Z(u16) Type(u8) = 7 bytes
  const entityX = new Uint16Array(numEntities);
  const entityY = new Uint16Array(numEntities);
  const entityZ = new Uint16Array(numEntities);
  const entityType = new Uint8Array(numEntities);

  for (let i = 0; i < numEntities; i++) {
    entityX[i] = view.getUint16(off, true);
    entityY[i] = view.getUint16(off + 2, true);
    entityZ[i] = view.getUint16(off + 4, true);
    entityType[i] = bytes[off + 6];
    off += 7;
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
    focusX,
    focusY,
    focusZ,
    focusRadius,
    cameraYaw,
    cameraPitch,
  };
}
