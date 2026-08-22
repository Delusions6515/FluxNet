import {
  enableEdgeToEdge,
  exec,
  getPackagesInfo,
  listPackages,
  moduleInfo,
} from "kernelsu";
import { MOCK_PACKAGES, mockGateway } from "./mock";

export const isBrowserDev = typeof window !== "undefined" && !window.ksu;
const info = isBrowserDev
  ? { version: "Browser Dev", moduleDir: "/data/adb/modules/fluxnet" }
  : (() => {
      try {
        return JSON.parse(moduleInfo());
      } catch {
        return {};
      }
    })();
export const MODULE_VERSION = info.version || "";
const script = `${info.moduleDir || "/data/adb/modules/fluxnet"}/scripts/webui.sh`;

export function requestEdgeToEdge() {
  if (!isBrowserDev) {
    try {
      enableEdgeToEdge(true);
    } catch {}
  }
}
function encode(value) {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (let offset = 0; offset < bytes.length; offset += 0x8000)
    binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
  return btoa(binary);
}
function quote(value) {
  return `'${String(value).replaceAll("'", "'\"'\"'")}'`;
}

async function gateway(command, args = []) {
  let raw;
  if (isBrowserDev) raw = mockGateway(command, args);
  else {
    const output = await exec(
      `sh ${quote(script)} ${quote(command)} ${args.map(quote).join(" ")}`,
    );
    if (output.errno !== 0)
      throw new Error(output.stderr || output.stdout || "模块命令执行失败");
    raw = output.stdout;
  }
  const response = JSON.parse(raw);
  if (response.schema !== 1) throw new Error("模块响应版本不受支持");
  if (!response.ok)
    throw new Error(response.message || response.code || "模块操作失败");
  return response.data;
}

export async function getOverview() {
  const [service, health, settings, subscriptions, logs] = await Promise.all(
    [
      "service-status",
      "health",
      "config-show",
      "subscription-list",
      "logs",
    ].map((command) => gateway(command)),
  );
  return { service, health, settings, subscriptions, logs: logs.entries || [] };
}
export const getSettings = () => gateway("config-show");
export const setSetting = (key, value) => gateway("config-set", [key, value]);
export const replaceAppList = (mode, apps) =>
  gateway("app-list-replace", [mode, encode(JSON.stringify(apps))]);
export const getSubscriptions = () => gateway("subscription-list");
export const addRemoteSubscription = (name, url) =>
  gateway("subscription-add-remote", [encode(name), encode(url)]);
export const updateSubscription = (name) =>
  gateway("subscription-update", [name]);
export const switchSubscription = (name) =>
  gateway("subscription-switch", [name]);
export const removeSubscription = (name) =>
  gateway("subscription-remove", [name]);
export const createLocalSubscription = (name) =>
  gateway("local-create", [name]);
export const readLocalSubscription = (name) => gateway("local-read", [name]);
export const writeLocalSubscription = (name, content) =>
  gateway("local-write", [name, encode(content)]);
export const serviceAction = (action) => gateway(`service-${action}`);

export async function getInstalledApps() {
  if (isBrowserDev) return MOCK_PACKAGES;
  try {
    const details = getPackagesInfo(listPackages("user"));
    if (details.length)
      return details.map(({ packageName, appLabel }) => ({
        packageName,
        appLabel: appLabel || packageName,
      }));
  } catch {}
  const output = await exec("pm list packages");
  return output.stdout
    .split("\n")
    .filter(Boolean)
    .map((line) => line.replace("package:", ""))
    .map((packageName) => ({ packageName, appLabel: packageName }));
}
