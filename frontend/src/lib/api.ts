import type { Video } from "../types/video";

const BASE =
  (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(
    /\/$/,
    "",
  ) ?? "";

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

// Partial metadata update (title / description / tags). Omitted fields are
// left unchanged on the server.
export async function updateVideo(
  id: string,
  patch: { title?: string; description?: string; tags?: string[] },
): Promise<Video> {
  const res = await fetch(`${BASE}/api/videos/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) {
    throw new Error(`updateVideo failed: ${res.status} ${await res.text()}`);
  }
  return res.json();
}

export async function deleteVideo(id: string): Promise<void> {
  const res = await fetch(`${BASE}/api/videos/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  if (!res.ok && res.status !== 204) {
    throw new Error(`deleteVideo failed: ${res.status} ${await res.text()}`);
  }
}

export async function uploadVideo(
  input: {
    file: File;
    thumbnail?: File | null;
    title?: string;
    description?: string;
    catName?: string;
    breed?: string;
    tags?: string[];
  },
  // Called with the upload fraction (0..1) as the file streams to the server,
  // so the UI can show a real progress bar instead of an indefinite block.
  onProgress?: (fraction: number) => void,
): Promise<Video> {
  const fd = new FormData();
  fd.append("file", input.file);
  if (input.thumbnail) fd.append("thumbnail", input.thumbnail);
  if (input.title) fd.append("title", input.title);
  if (input.description) fd.append("description", input.description);
  if (input.catName) fd.append("catName", input.catName);
  if (input.breed) fd.append("breed", input.breed);
  if (input.tags && input.tags.length) fd.append("tags", input.tags.join(","));

  // XMLHttpRequest (not fetch) because only XHR exposes upload progress events.
  return new Promise<Video>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `${BASE}/api/videos`);
    xhr.upload.onprogress = (e) => {
      if (onProgress && e.lengthComputable) onProgress(e.loaded / e.total);
    };
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText) as Video);
        } catch {
          reject(new Error("uploadVideo: invalid server response"));
        }
      } else {
        reject(new Error(`uploadVideo failed: ${xhr.status}`));
      }
    };
    xhr.onerror = () => reject(new Error("uploadVideo failed: network error"));
    xhr.onabort = () => reject(new Error("uploadVideo aborted"));
    xhr.send(fd);
  });
}
