import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import type { PreviewData } from "./decoder";

// Entity type colors
const ENTITY_COLORS: Record<number, number> = {
  0: 0x00ff00, // PlayerStart - green
  1: 0xff0000, // Flag - red
  2: 0xffff00, // Base - yellow
  3: 0x00ffff, // Teleport - cyan
  4: 0xff00ff, // JumpPad - magenta
  5: 0xff6600, // Health - orange
  6: 0x3366ff, // Armour - blue
  7: 0xcccc00, // Ammo - gold
  8: 0xff00ff, // Quad - purple
};

// Extract voxel size from flags bits 5-7: size = 1 << ((flags >> 5) & 7)
function voxelSize(flags: number): number {
  return 1 << ((flags >> 5) & 7);
}

export function createScene(
  canvas: HTMLCanvasElement,
  data: PreviewData,
  width: number,
  height: number
) {
  const renderer = new THREE.WebGLRenderer({
    canvas,
    antialias: true,
    alpha: true,
  });
  renderer.setSize(width, height);
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.5));
  renderer.localClippingEnabled = true;

  const scene = new THREE.Scene();

  // Sky gradient background
  scene.background = new THREE.Color(
    data.skyTop[0] / 255,
    data.skyTop[1] / 255,
    data.skyTop[2] / 255
  );

  // Focus point and orbit distance from file
  // Sauer (X, Y, Z) → Three.js (X, Z, Y)
  const gridSize = data.gridSize;
  const focusX = data.focusX;
  const focusY = data.focusZ; // Sauer Z → Three Y
  const focusZ = data.focusY; // Sauer Y → Three Z
  const orbitRadius = data.focusRadius;

  // Camera
  const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, gridSize * 10);
  camera.position.set(
    focusX + orbitRadius * 0.8,
    focusY + orbitRadius * 0.5,
    focusZ + orbitRadius * 0.8
  );

  // Lighting — bright enough to show texture colors clearly
  const ambientLight = new THREE.AmbientLight(0xffffff, 0.85);
  scene.add(ambientLight);

  const sunLight = new THREE.DirectionalLight(0xffffff, 0.8);
  sunLight.position.set(gridSize * 1.5, gridSize * 2, gridSize * 0.5);
  scene.add(sunLight);

  const fillLight = new THREE.DirectionalLight(0x8899bb, 0.3);
  fillLight.position.set(-gridSize, gridSize * 0.5, -gridSize);
  scene.add(fillLight);

  // Clipping plane for cross-section
  const clipPlane = new THREE.Plane(new THREE.Vector3(0, -1, 0), gridSize);

  // Voxels via InstancedMesh
  const boxGeo = new THREE.BoxGeometry(1, 1, 1);
  const voxelMat = new THREE.MeshLambertMaterial({
    vertexColors: true,
    clippingPlanes: [clipPlane],
    clipShadows: true,
  });

  const numVoxels = data.numVoxels;
  const mesh = new THREE.InstancedMesh(boxGeo, voxelMat, numVoxels);

  const dummy = new THREE.Object3D();
  const colors = new Float32Array(numVoxels * 3);

  for (let i = 0; i < numVoxels; i++) {
    const size = voxelSize(data.voxelFlags[i]);
    const halfSize = (size - 1) * 0.5;

    // Position: corner + half-size offset to center the scaled cube
    // Sauerbraten (x, y, z) → Three.js (x, z, y)
    dummy.position.set(
      data.voxelX[i] + halfSize,
      data.voxelZ[i] + halfSize,
      data.voxelY[i] + halfSize
    );
    dummy.scale.set(size, size, size);
    dummy.updateMatrix();
    mesh.setMatrixAt(i, dummy.matrix);

    // Color from palette with subtle AO
    const palIdx = data.voxelColor[i] * 3;
    // AO: soften the effect so colors stay visible
    // ao=0 → 0.4 brightness, ao=255 → 1.0 brightness
    const ao = 0.4 + 0.6 * (data.voxelAO[i] / 255);
    colors[i * 3] = (data.palette[palIdx] / 255) * ao;
    colors[i * 3 + 1] = (data.palette[palIdx + 1] / 255) * ao;
    colors[i * 3 + 2] = (data.palette[palIdx + 2] / 255) * ao;
  }

  mesh.instanceMatrix.needsUpdate = true;
  mesh.geometry.setAttribute(
    "color",
    new THREE.InstancedBufferAttribute(colors, 3)
  );
  scene.add(mesh);

  // Entity markers — scale with grid
  if (data.numEntities > 0) {
    const sphereGeo = new THREE.SphereGeometry(1.2, 8, 8);
    const entityMat = new THREE.MeshBasicMaterial({
      clippingPlanes: [clipPlane],
    });

    for (let i = 0; i < data.numEntities; i++) {
      const mat = entityMat.clone();
      mat.color.setHex(ENTITY_COLORS[data.entityType[i]] ?? 0xffffff);
      const sphere = new THREE.Mesh(sphereGeo, mat);
      sphere.position.set(data.entityX[i], data.entityZ[i], data.entityY[i]);
      scene.add(sphere);
    }
  }

  // OrbitControls
  const controls = new OrbitControls(camera, canvas);
  controls.target.set(focusX, focusY, focusZ);
  controls.enableDamping = true;
  controls.dampingFactor = 0.1;
  controls.autoRotate = true;
  controls.autoRotateSpeed = 1.5;
  controls.minDistance = 5;
  controls.maxDistance = gridSize * 4;
  controls.update();

  // Animation loop
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
