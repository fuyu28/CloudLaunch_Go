/**
 * @fileoverview DaisyUIテーマ関連の定数
 *
 * DaisyUIで利用可能なテーマの一覧を定義します。
 */

export const DAISYUI_THEMES = [
  "cloudlaunch",
  "light",
  "dark",
  "cupcake",
  "bumblebee",
  "emerald",
  "corporate",
  "synthwave",
  "retro",
  "cyberpunk",
  "valentine",
  "halloween",
  "garden",
  "forest",
  "aqua",
  "lofi",
  "pastel",
  "fantasy",
  "wireframe",
  "black",
  "luxury",
  "dracula",
  "cmyk",
  "autumn",
  "business",
  "acid",
  "lemonade",
  "night",
  "coffee",
  "winter",
  "dim",
  "nord",
  "sunset",
  "caramellatte",
  "abyss",
  "silk",
] as const;

export type ThemeName = (typeof DAISYUI_THEMES)[number];
