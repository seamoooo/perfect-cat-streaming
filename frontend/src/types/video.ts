export type Breed = "siamese" | "bengal" | "other";

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
