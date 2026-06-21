// Breed slug from the CAT_BREEDS list (lib/breeds.ts). Open string since the
// list can grow without touching this type.
export type Breed = string;

export type VideoStatus = "pending" | "processing" | "ready" | "error";

export interface Video {
  id: string;
  title: string;
  description: string;
  catName: string;
  breed: Breed;
  tags: string[];
  durationSec: number;
  status: VideoStatus;
  errorMsg?: string;
  playlistUrl?: string;
  createdAt: string;
  updatedAt: string;
}
