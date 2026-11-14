export type Report = {
  id: string;
  authorId: string;
  authorName: string;
  kind: "bug" | "feature";
  title: string;
  body?: string;
  status: "submitted" | "accepted" | "denied" | "closed";
  release: string;
  releaseURL?: string;
  hasScreenshot: boolean;
  createdAt: number;
  updatedAt: number;
  comments: {
    id: string;
    authorId: string;
    authorName: string;
    body: string;
    createdAt: number;
  }[];
};
