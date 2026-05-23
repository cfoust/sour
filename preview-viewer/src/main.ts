import { decodePreview, type PreviewData } from "./decoder";
import { createScene } from "./scene";

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

  document.querySelectorAll(".map-item").forEach((el) => {
    el.classList.toggle("active", el.getAttribute("data-map") === name);
  });

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
    const kb = Math.round(((await fetch(`/${name}.svox`)).headers.get("content-length") as any) / 1024);
    stats.textContent = `${data.numVoxels.toLocaleString()} voxels | ${kb} KB`;

    slider.value = "100";
    slider.oninput = () => {
      currentScene?.setClipHeight(parseInt(slider.value) / 100);
    };
  } catch (e) {
    stats.textContent = `Failed to load ${name}`;
    console.error(e);
  }
}

// Discover all .svox files
async function discoverMaps(): Promise<string[]> {
  const resp = await fetch("/maps.json");
  if (resp.ok) {
    return await resp.json();
  }
  // Fallback: try a known set
  return ["complex", "dust2", "turbine"];
}

async function init() {
  const app = document.getElementById("app")!;
  app.innerHTML = `
    <nav id="sidebar">
      <input type="text" id="search" placeholder="Search maps..." />
      <div id="map-list"></div>
    </nav>
    <main id="main">
      <canvas id="viewer"></canvas>
      <div id="hud">
        <span id="stats"></span>
        <input type="range" id="clip-slider" min="0" max="100" value="100" />
      </div>
    </main>
  `;

  const maps = await discoverMaps();
  const mapList = document.getElementById("map-list")!;
  const search = document.getElementById("search") as HTMLInputElement;

  function renderList(filter: string) {
    mapList.innerHTML = "";
    const filtered = filter
      ? maps.filter((m) => m.toLowerCase().includes(filter.toLowerCase()))
      : maps;
    for (const name of filtered) {
      const item = document.createElement("div");
      item.className = "map-item" + (name === activeMap ? " active" : "");
      item.setAttribute("data-map", name);
      item.textContent = name;
      item.onclick = () => selectMap(name);
      mapList.appendChild(item);
    }
  }

  search.oninput = () => renderList(search.value);
  renderList("");

  if (maps.length > 0) {
    selectMap(maps[0]);
  }
}

init();
