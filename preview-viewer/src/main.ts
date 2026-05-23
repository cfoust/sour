import { decodePreview, type PreviewData } from "./decoder";
import { createScene } from "./scene";

const MAPS = [
  "complex", "dust2", "turbine",
  "akaritori", "alithia", "arabic", "caribbean", "castle_trap", "curvedm",
  "duel8", "fc5", "flagstone", "hog2", "justice", "lost",
  "neondevastation", "neonpanic", "ot", "reissen", "ruby", "suburb",
];

const cache = new Map<string, PreviewData>();
let currentScene: ReturnType<typeof createScene> | null = null;
let activeMap = "";

async function loadPreview(name: string): Promise<PreviewData> {
  if (cache.has(name)) return cache.get(name)!;
  const response = await fetch(`/${name}.svox`);
  if (!response.ok) throw new Error(`Failed to load ${name}.svox`);
  const data = decodePreview(await response.arrayBuffer());
  cache.set(name, data);
  return data;
}

async function selectMap(name: string) {
  if (name === activeMap) return;
  activeMap = name;

  // Update sidebar selection
  document.querySelectorAll(".map-item").forEach((el) => {
    el.classList.toggle("active", el.getAttribute("data-map") === name);
  });

  // Dispose old scene
  if (currentScene) {
    currentScene.dispose();
    currentScene = null;
  }

  const canvas = document.getElementById("viewer") as HTMLCanvasElement;
  const stats = document.getElementById("stats")!;
  const slider = document.getElementById("clip-slider") as HTMLInputElement;
  stats.textContent = "Loading...";

  try {
    const data = await loadPreview(name);
    const rect = canvas.parentElement!.getBoundingClientRect();
    const size = Math.min(rect.width, rect.height);
    canvas.width = size;
    canvas.height = size;

    currentScene = createScene(canvas, data, size, size);
    stats.textContent = `${data.numVoxels.toLocaleString()} voxels, ${data.gridSize}\u00B3 grid`;

    slider.value = "100";
    slider.oninput = () => {
      currentScene?.setClipHeight(parseInt(slider.value) / 100);
    };
  } catch (e) {
    stats.textContent = `Failed to load ${name}`;
    console.error(e);
  }
}

// Build UI
const app = document.getElementById("app")!;
app.innerHTML = `
  <nav id="sidebar"></nav>
  <main id="main">
    <canvas id="viewer"></canvas>
    <div id="hud">
      <span id="stats"></span>
      <input type="range" id="clip-slider" min="0" max="100" value="100" />
    </div>
  </main>
`;

const sidebar = document.getElementById("sidebar")!;
for (const name of MAPS) {
  const item = document.createElement("div");
  item.className = "map-item";
  item.setAttribute("data-map", name);
  item.textContent = name;
  item.onclick = () => selectMap(name);
  sidebar.appendChild(item);
}

selectMap(MAPS[0]);
