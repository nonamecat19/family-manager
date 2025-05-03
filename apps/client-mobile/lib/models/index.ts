export interface ListItem {
  id: string;
  title: string;
  completed?: boolean;
  categoryId?: string;
  tagIds?: string[];
}

export interface Category {
  id: string;
  name: string;
}

export interface Tag {
  id: string;
  name: string;
}