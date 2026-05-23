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
  const topColor = new THREE.Color(
    data.skyTop[0] / 255,
    data.skyTop[1] / 255,
    data.skyTop[2] / 255
  );
  const bottomColor = new THREE.Color(
    data.skyHorizon[0] / 255,
    data.skyHorizon[1] / 255,
    data.skyHorizon[2] / 255
  );
  scene.background = topColor;

  // Camera
  const gridSize = data.gridSize;
  const center = gridSize / 2;
  const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 500);
  camera.position.set(center + gridSize, center + gridSize * 0.6, center + gridSize);
  camera.lookAt(center, center * 0.3, center);

  // Lighting
  const ambientLight = new THREE.AmbientLight(
    new THREE.Color(
      data.ambient[0] / 255,
      data.ambient[1] / 255,
      data.ambient[2] / 255
    ),
    0.6
  );
  scene.add(ambientLight);

  const sunLight = new THREE.DirectionalLight(
    new THREE.Color(
      data.sunlight[0] / 255,
      data.sunlight[1] / 255,
      data.sunlight[2] / 255
    ),
    1.2
  );
  sunLight.position.set(gridSize * 1.5, gridSize * 2, gridSize * 0.5);
  scene.add(sunLight);

  const fillLight = new THREE.DirectionalLight(0x4466aa, 0.3);
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
    // Position: Sauerbraten uses (x, z, y) -> Three.js (x, y, z)
    // Map Y (up in Sauer) to Y (up in Three.js)
    dummy.position.set(data.voxelX[i], data.voxelZ[i], data.voxelY[i]);
    dummy.updateMatrix();
    mesh.setMatrixAt(i, dummy.matrix);

    // Color from palette with AO
    const palIdx = data.voxelColor[i] * 3;
    const ao = data.voxelAO[i] / 255;
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

  // Entity markers
  if (data.numEntities > 0) {
    const sphereGeo = new THREE.SphereGeometry(0.5, 6, 6);
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
  controls.target.set(center, center * 0.3, center);
  controls.enableDamping = true;
  controls.dampingFactor = 0.1;
  controls.autoRotate = true;
  controls.autoRotateSpeed = 1.5;
  controls.minDistance = 5;
  controls.maxDistance = gridSize * 3;
  controls.update();

  // Animation loop
  let animationId: number;
  let lastTime = 0;
  const frameInterval = 1000 / 30; // 30fps cap

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
      // normalized 0-1, where 1 = show all, 0 = show nothing
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
