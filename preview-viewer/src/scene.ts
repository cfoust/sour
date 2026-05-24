import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import type { PreviewData } from "./decoder";

const ENTITY_COLORS: Record<number, number> = {
  0: 0x00ff00, 1: 0xff0000, 2: 0xffff00, 3: 0x00ffff,
  4: 0xff00ff, 5: 0xff6600, 6: 0x3366ff, 7: 0xcccc00, 8: 0xff00ff,
};

function voxelSize(flags: number): number {
  return 1 << ((flags >> 5) & 7);
}

// Simple deterministic hash for per-voxel variation
function hash3(x: number, y: number, z: number): number {
  let h = (x * 374761393 + y * 668265263 + z * 1274126177) | 0;
  h = ((h ^ (h >> 13)) * 1031) | 0;
  return ((h ^ (h >> 16)) & 0xffff) / 65535;
}

export function createScene(
  canvas: HTMLCanvasElement,
  data: PreviewData,
  width: number,
  height: number
) {
  const renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: true });
  renderer.setSize(width, height);
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
  renderer.localClippingEnabled = true;

  const scene = new THREE.Scene();
  scene.background = new THREE.Color(
    data.skyTop[0] / 255, data.skyTop[1] / 255, data.skyTop[2] / 255
  );

  const gridSize = data.gridSize;
  const focusX = data.focusX;
  const focusY = data.focusZ;
  const focusZ = data.focusY;
  const orbitRadius = data.focusRadius;

  const yawRad = (data.cameraYaw * Math.PI) / 180;
  const pitchRad = (data.cameraPitch * Math.PI) / 180;
  const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, gridSize * 10);
  camera.position.set(
    focusX + Math.cos(pitchRad) * Math.cos(yawRad) * orbitRadius,
    focusY + Math.sin(pitchRad) * orbitRadius,
    focusZ + Math.cos(pitchRad) * Math.sin(yawRad) * orbitRadius
  );

  // Lighting — hemisphere for natural sky/ground color variation
  const hemiLight = new THREE.HemisphereLight(0x8899cc, 0x554433, 0.5);
  scene.add(hemiLight);

  // Strong directional sun for shadows and depth
  const sunLight = new THREE.DirectionalLight(0xffeedd, 1.2);
  sunLight.position.set(gridSize * 1.5, gridSize * 2, gridSize * 0.5);
  scene.add(sunLight);

  // Cool fill from the opposite side
  const fillLight = new THREE.DirectionalLight(0x6688aa, 0.4);
  fillLight.position.set(-gridSize, gridSize * 0.3, -gridSize);
  scene.add(fillLight);

  // Clip plane
  const defaultClip = data.clipY > 0 && data.clipY < gridSize ? data.clipY : gridSize;
  const clipPlane = new THREE.Plane(new THREE.Vector3(0, -1, 0), defaultClip);

  // Voxels
  const boxGeo = new THREE.BoxGeometry(1, 1, 1);
  const voxelMat = new THREE.MeshLambertMaterial({
    vertexColors: true,
    clippingPlanes: [clipPlane],
    clipShadows: true,
  });

  // Separate solid, water, and lava voxels
  const MAT_WATER = 0x04; // FlagWater = 1 << 2
  const MAT_LAVA  = 0x08; // FlagLava  = 2 << 2
  const MAT_MASK  = 0x1c; // bits 2-4
  let solidCount = 0;
  let waterCount = 0;
  let lavaCount = 0;
  for (let i = 0; i < data.numVoxels; i++) {
    const mat = data.voxelFlags[i] & MAT_MASK;
    if (mat === MAT_WATER) waterCount++;
    else if (mat === MAT_LAVA) lavaCount++;
    else solidCount++;
  }

  // Solid voxels
  const mesh = new THREE.InstancedMesh(boxGeo, voxelMat, solidCount);
  const dummy = new THREE.Object3D();
  const colors = new Float32Array(solidCount * 3);

  // Water/lava cubes — shrunk 2% inward to avoid z-fighting with adjacent solids
  const waterMat = new THREE.MeshLambertMaterial({
    color: 0x2266cc,
    transparent: true,
    opacity: 0.5,
    depthWrite: false,
    polygonOffset: true,
    polygonOffsetFactor: -2,
    polygonOffsetUnits: -2,
    clippingPlanes: [clipPlane],
  });
  const waterMesh = new THREE.InstancedMesh(boxGeo, waterMat, Math.max(waterCount, 1));

  const lavaMat = new THREE.MeshBasicMaterial({
    color: 0xff4400,
    transparent: true,
    opacity: 0.9,
    depthWrite: false,
    polygonOffset: true,
    polygonOffsetFactor: -2,
    polygonOffsetUnits: -2,
    clippingPlanes: [clipPlane],
  });
  const lavaMesh = new THREE.InstancedMesh(boxGeo, lavaMat, Math.max(lavaCount, 1));

  let solidIdx = 0;
  let waterIdx = 0;
  let lavaIdx = 0;

  for (let i = 0; i < data.numVoxels; i++) {
    const size = voxelSize(data.voxelFlags[i]);
    const halfSize = (size - 1) * 0.5;
    const vx = data.voxelX[i];
    const vy = data.voxelY[i];
    const vz = data.voxelZ[i];
    const matBits = data.voxelFlags[i] & MAT_MASK;

    dummy.position.set(vx + halfSize, vz + halfSize, vy + halfSize);
    dummy.scale.set(size, size, size);
    dummy.updateMatrix();

    if (matBits === MAT_WATER || matBits === MAT_LAVA) {
      // Shrink slightly inward to avoid coplanar faces with solid geometry
      const shrink = size * 0.98;
      dummy.position.set(vx + halfSize, vz + halfSize, vy + halfSize);
      dummy.scale.set(shrink, shrink, shrink);
      dummy.updateMatrix();
      if (matBits === MAT_WATER) {
        waterMesh.setMatrixAt(waterIdx++, dummy.matrix);
      } else {
        lavaMesh.setMatrixAt(lavaIdx++, dummy.matrix);
      }
    } else {
      mesh.setMatrixAt(solidIdx, dummy.matrix);

      const palIdx = data.voxelColor[i] * 3;
      let r = data.palette[palIdx] / 255;
      let g = data.palette[palIdx + 1] / 255;
      let b = data.palette[palIdx + 2] / 255;

      const lum = 0.299 * r + 0.587 * g + 0.114 * b;
      const satBoost = 2.2;
      r = lum + (r - lum) * satBoost;
      g = lum + (g - lum) * satBoost;
      b = lum + (b - lum) * satBoost;
      const lift = 1.3;
      r = Math.max(0, Math.min(1, r * lift));
      g = Math.max(0, Math.min(1, g * lift));
      b = Math.max(0, Math.min(1, b * lift));

      const ao = 0.2 + 0.8 * (data.voxelAO[i] / 255);
      const jitter = 0.85 + 0.3 * hash3(vx, vy, vz);

      colors[solidIdx * 3] = Math.min(r * ao * jitter, 1);
      colors[solidIdx * 3 + 1] = Math.min(g * ao * jitter, 1);
      colors[solidIdx * 3 + 2] = Math.min(b * ao * jitter, 1);
      solidIdx++;
    }
  }

  mesh.instanceMatrix.needsUpdate = true;
  mesh.geometry.setAttribute("color", new THREE.InstancedBufferAttribute(colors, 3));
  scene.add(mesh);

  if (waterCount > 0) {
    waterMesh.instanceMatrix.needsUpdate = true;
    waterMesh.renderOrder = 1;
    scene.add(waterMesh);
  }
  if (lavaCount > 0) {
    lavaMesh.instanceMatrix.needsUpdate = true;
    lavaMesh.renderOrder = 1;
    scene.add(lavaMesh);
  }

  // Entity markers
  if (data.numEntities > 0) {
    const sphereGeo = new THREE.SphereGeometry(1.2, 8, 8);
    for (let i = 0; i < data.numEntities; i++) {
      const mat = new THREE.MeshBasicMaterial({
        color: ENTITY_COLORS[data.entityType[i]] ?? 0xffffff,
        clippingPlanes: [clipPlane],
      });
      const sphere = new THREE.Mesh(sphereGeo, mat);
      sphere.position.set(data.entityX[i], data.entityZ[i], data.entityY[i]);
      scene.add(sphere);
    }
  }

  // Controls
  const controls = new OrbitControls(camera, canvas);
  controls.target.set(focusX, focusY, focusZ);
  controls.enableDamping = true;
  controls.dampingFactor = 0.1;
  controls.autoRotate = true;
  controls.autoRotateSpeed = 1.5;
  controls.minDistance = 5;
  controls.maxDistance = gridSize * 4;
  controls.update();

  let animationId: number;
  let lastTime = 0;
  const frameInterval = 1000 / 30;

  function animate(time: number) {
    animationId = requestAnimationFrame(animate);
    const delta = time - lastTime;
    if (delta < frameInterval) return;
    lastTime = time - (delta % frameInterval);
    controls.update();
    renderer.render(scene, camera);
  }
  animationId = requestAnimationFrame(animate);

  return {
    setClipHeight(normalized: number) {
      clipPlane.constant = normalized * gridSize;
    },
    dispose() {
      cancelAnimationFrame(animationId);
      controls.dispose();
      renderer.dispose();
      boxGeo.dispose();
      voxelMat.dispose();
    },
  };
}
