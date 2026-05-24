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
  const focusY = data.focusZ; // Sauer Z → Three Y
  const focusZ = data.focusY; // Sauer Y → Three Z
  const orbitRadius = data.focusRadius;

  // Camera from baked angle
  const yawRad = (data.cameraYaw * Math.PI) / 180;
  const pitchRad = (data.cameraPitch * Math.PI) / 180;
  const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, gridSize * 10);
  camera.position.set(
    focusX + Math.cos(pitchRad) * Math.cos(yawRad) * orbitRadius,
    focusY + Math.sin(pitchRad) * orbitRadius,
    focusZ + Math.cos(pitchRad) * Math.sin(yawRad) * orbitRadius
  );

  // Lighting
  scene.add(new THREE.AmbientLight(0xffffff, 0.85));
  const sunLight = new THREE.DirectionalLight(0xffffff, 0.8);
  sunLight.position.set(gridSize * 1.5, gridSize * 2, gridSize * 0.5);
  scene.add(sunLight);
  const fillLight = new THREE.DirectionalLight(0x8899bb, 0.3);
  fillLight.position.set(-gridSize, gridSize * 0.5, -gridSize);
  scene.add(fillLight);

  // Flat clip plane — default from baked ClipY, adjustable via slider
  // ClipY is in Sauer Z coords → Three Y
  const defaultClip = data.clipY > 0 && data.clipY < gridSize ? data.clipY : gridSize;
  const clipPlane = new THREE.Plane(new THREE.Vector3(0, -1, 0), defaultClip);

  // Voxels
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
    dummy.position.set(
      data.voxelX[i] + halfSize,
      data.voxelZ[i] + halfSize,
      data.voxelY[i] + halfSize
    );
    dummy.scale.set(size, size, size);
    dummy.updateMatrix();
    mesh.setMatrixAt(i, dummy.matrix);

    const palIdx = data.voxelColor[i] * 3;
    const ao = 0.4 + 0.6 * (data.voxelAO[i] / 255);
    colors[i * 3] = (data.palette[palIdx] / 255) * ao;
    colors[i * 3 + 1] = (data.palette[palIdx + 1] / 255) * ao;
    colors[i * 3 + 2] = (data.palette[palIdx + 2] / 255) * ao;
  }

  mesh.instanceMatrix.needsUpdate = true;
  mesh.geometry.setAttribute("color", new THREE.InstancedBufferAttribute(colors, 3));
  scene.add(mesh);

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
      // Slider 0-100 maps to 0 - gridSize
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
