// Drives the real running app (frontend :3000, backend :8080, anvil
// :8545 — all from the previous lessons, already up) with headless Chrome
// to produce PNG frames, then shells out to ffmpeg to turn each lesson's
// frames into docs/gifs/lesson-NN.gif. Pure documentation tooling, not
// part of the shipped app. See docs/lesson-*.md "ทดสอบเอง" for how to get
// the prerequisite processes running first.
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import puppeteer from "puppeteer-core";

const ROOT = path.resolve(import.meta.dirname, "..", "..");
const FRAMES_DIR = path.join(ROOT, "docs", "media", "_frames");
const GIFS_DIR = path.join(ROOT, "docs", "gifs");
const SCRIPTS_DIR = import.meta.dirname;
const CHROME_PATHS = ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"];

fs.mkdirSync(FRAMES_DIR, { recursive: true });
fs.mkdirSync(GIFS_DIR, { recursive: true });

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

let shotIndex = {};
async function shot(page, lesson, name) {
  shotIndex[lesson] = (shotIndex[lesson] || 0) + 1;
  const file = path.join(FRAMES_DIR, `lesson${lesson}-${String(shotIndex[lesson]).padStart(2, "0")}-${name}.png`);
  await page.screenshot({ path: file });
  console.log("captured", file);
}

// --- terminal-transcript lessons (01, 04, 05, 07): replay real command
// output that was captured while building the lesson.
const TERMINAL_LESSON_STEPS = { "01": 9, "04": 8, "05": 11, "07": 11 };

async function captureTerminalLessons(browser) {
  for (const [lesson, steps] of Object.entries(TERMINAL_LESSON_STEPS)) {
    const page = await browser.newPage();
    await page.setViewport({ width: 948, height: 560 });
    const fileUrl = "file://" + path.join(SCRIPTS_DIR, `terminal-lesson-${lesson}.html`);
    for (let step = 1; step <= steps; step++) {
      await page.goto(`${fileUrl}?step=${step}`, { waitUntil: "networkidle0" });
      await shot(page, lesson, `step${step}`);
    }
    await page.close();
  }
}

async function renderDiagramPng(browser) {
  const page = await browser.newPage();
  const svgPath = path.join(ROOT, "docs", "diagrams", "07-awal-autonomous-flow.svg");
  const pngPath = path.join(ROOT, "docs", "diagrams", "07-awal-autonomous-flow.png");
  if (fs.existsSync(svgPath)) {
    const svgContent = fs.readFileSync(svgPath, "utf8");
    await page.setContent(`<!doctype html><html><body style="margin:0;padding:20px;background:#fff;">${svgContent}</body></html>`);
    const el = await page.$("svg");
    if (el) {
      await el.screenshot({ path: pngPath });
      console.log("rendered diagram PNG:", pngPath);
    }
  }
  await page.close();
}

// --- UI lessons (02, 03, 06): drive the real running app end-to-end,
// slicing the one continuous journey into per-lesson frame prefixes.
async function captureUiLessons(browser) {
  const page = await browser.newPage();
  await page.setViewport({ width: 1000, height: 800 });

  // lesson 02: catalog -> book -> cart
  await page.goto("http://localhost:3000/", { waitUntil: "networkidle0" });
  await shot(page, "02", "home");

  const firstBook = await page.waitForSelector('a[href^="/book/"]');
  await firstBook.click();
  await page.waitForNetworkIdle({ idleTime: 300 });
  await shot(page, "02", "book-detail");

  const addBtn = await page.waitForSelector("button");
  await addBtn.click();
  await page.waitForFunction(() => location.pathname.startsWith("/cart/"), { timeout: 10000 });
  await sleep(300);
  await shot(page, "02", "cart");

  // lesson 03: order page, AP2 mandate chain steps 1-3
  const checkoutBtn = await page.waitForSelector("button");
  await checkoutBtn.click();
  await page.waitForFunction(() => location.pathname.startsWith("/order/"), { timeout: 10000 });
  await sleep(300);
  await shot(page, "03", "order-empty");

  async function clickButtonContaining(text) {
    const [btn] = await page.$$("xpath/.//button[contains(., '" + text + "')]");
    if (!btn) throw new Error(`button containing "${text}" not found`);
    await btn.click();
    await sleep(1200);
  }

  await clickButtonContaining("สร้าง Intent Mandate");
  await shot(page, "03", "intent-mandate");

  await clickButtonContaining("สร้าง Cart Mandate");
  await shot(page, "03", "cart-mandate");

  await clickButtonContaining("อนุมัติจ่ายเงิน");
  await shot(page, "03", "payment-mandate");

  // lesson 06: agent self-checkout, on-chain, final paid state
  await shot(page, "06", "ready-to-pay");
  await clickButtonContaining("ให้ agent จ่ายเงิน");
  await sleep(3500); // real on-chain tx: approve + pay + verify
  await shot(page, "06", "paid");

  await page.close();
}

async function assembleGifs() {
  const files = fs.readdirSync(FRAMES_DIR);
  const lessons = [...new Set(files.map((f) => f.match(/^lesson(\d+)-/)?.[1]).filter(Boolean))];
  for (const lesson of lessons) {
    const pattern = path.join(FRAMES_DIR, `lesson${lesson}-*.png`);
    const out = path.join(GIFS_DIR, `lesson-${lesson}.gif`);
    execFileSync("ffmpeg", [
      "-y",
      "-framerate", "0.8",
      "-pattern_type", "glob",
      "-i", pattern,
      "-vf", "scale=760:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse",
      "-loop", "0",
      out,
    ]);
    console.log("wrote", out);
  }
}

async function main() {
  const executablePath = CHROME_PATHS.find((p) => fs.existsSync(p));
  if (!executablePath) throw new Error("Chrome not found at " + CHROME_PATHS.join(", "));
  const browser = await puppeteer.launch({ executablePath, headless: true });

  await renderDiagramPng(browser);
  await captureTerminalLessons(browser);
  try {
    await captureUiLessons(browser);
  } catch (err) {
    console.warn("UI lessons capture skipped:", err.message);
  }

  await browser.close();
  await assembleGifs();
  console.log("done. frames in", FRAMES_DIR, "gifs in", GIFS_DIR);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
