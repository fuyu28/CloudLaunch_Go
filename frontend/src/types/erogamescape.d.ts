export type ErogameScapeSearchItem = {
  erogameScapeId: string;
  title: string;
  brand?: string;
  gameUrl: string;
};

export type ErogameScapeSearchResult = {
  items: ErogameScapeSearchItem[];
  nextPageUrl?: string;
};
