/** The editor's single door. */
export {
  NO_BLOCK,
  canRetype,
  makeBlock,
  ensureBlocks,
  newBlockId,
  replaceAt,
  splitAt,
  mergeBackwards,
  insertAfter,
  insertDivider,
  moveBlock,
  setType,
  toggleChecked,
  applyMarkdownPrefix,
  outline,
  isEmptyDocument,
} from "./blocks.ts";

export {
  BlockEditor,
  BLOCK_COLUMN_WIDTH,
  GUTTER_LANE,
  type BlockEditorProps,
} from "./BlockEditor.tsx";
export { BlockBar, type BlockBarProps } from "./BlockBar.tsx";
export { useNoteDraft, SAVE_DEBOUNCE_MS, type NoteDraft, type SaveState } from "./useNoteDraft.ts";
