import type { Video } from "../types/video";

const BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") ?? "";

export async function listVideos(): Promise<Video[]> {
  const res = await fetch(`${BASE}/api/videos`);
  if (!res.ok) throw new Error(`listVideos failed: ${res.status}`);
  return res.json();
}

export async function getVideo(id: string): Promise<Video> {
  const res = await fetch(`${BASE}/api/videos/${id}`);
  if (!res.ok) throw new Error(`getVideo failed: ${res.status}`);
  return res.json();
}

export async function uploadVideo(input: {
  file: File;
  title?: string;
  description?: string;
  catName?: string;
  breed?: "siamese" | "bengal" | "other";
  tags?: string[];
}): Promise<Video> {
  const fd = new FormData();
  fd.append("file", input.file);
  if (input.title) fd.append("title", input.title);
  if (input.description) fd.append("description", input.description);
  if (input.catName) fd.append("catName", input.catName);
  if (input.breed) fd.append("breed", input.breed);
  if (input.tags && input.tags.length) fd.append("tags", input.tags.join(","));
  const res = await fetch(`${BASE}/api/videos`, { method: "POST", body: fd });
  if (!res.ok) throw new Error(`uploadVideo failed: ${res.status}`);
  return res.json();
}
