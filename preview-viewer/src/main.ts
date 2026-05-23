import { decodePreview } from "./decoder";
import { createScene } from "./scene";

const MAPS = ["complex", "dust2", "turbine"];
const CARD_SIZE = 480;

async function loadPreview(name: string) {
  const response = await fetch(`/${name}.svox`);
  if (!response.ok) {
    throw new Error(`Failed to load ${name}.svox: ${response.statusText}`);
  }
  const buffer = await response.arrayBuffer();
  return decodePreview(buffer);
}

async function createCard(name: string) {
  const app = document.getElementById("app")!;

  const card = document.createElement("div");
  card.className = "card";

  const canvas = document.createElement("canvas");
  canvas.width = CARD_SIZE;
  canvas.height = CARD_SIZE;
  card.appendChild(canvas);

  const label = document.createElement("div");
  label.className = "label";
  label.textContent = name;
  card.appendChild(label);

  const controlsDiv = document.createElement("div");
  controlsDiv.className = "controls";

  const slider = document.createElement("input");
  slider.type = "range";
  slider.className = "clip-slider";
  slider.min = "0";
  slider.max = "100";
  slider.value = "100";
  controlsDiv.appendChild(slider);
  card.appendChild(controlsDiv);

  app.appendChild(card);

  try {
    const data = await loadPreview(name);

    const stats = document.createElement("div");
    stats.className = "stats";
    stats.textContent = `${data.numVoxels} voxels, ${data.gridSize}³ grid`;
    card.appendChild(stats);

    const scene = createScene(canvas, data, CARD_SIZE, CARD_SIZE);

    slider.addEventListener("input", () => {
      scene.setClipHeight(parseInt(slider.value) / 100);
    });
  } catch (e) {
    console.error(`Failed to load ${name}:`, e);
    label.textContent = `${name} (failed to load)`;
  }
}

// Load all maps
for (const name of MAPS) {
  createCard(name);
}
