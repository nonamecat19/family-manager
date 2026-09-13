import assert from "node:assert/strict";
import { test } from "node:test";

import { blocksToPlainText, countTasks, type BlockLike } from "./noteBlocks.ts";
import { normalizeNoteFilters, queryKeys } from "./queryKeys.ts";

const BlockType = { HEADING: 2, TODO: 3, BULLET: 4, PARAGRAPH: 1, IMAGE: 8, DIVIDER: 9 } as const;
const NoteSort = { UNSPECIFIED: 0, TITLE: 3 } as const;
const SearchFacet = { ALL: 1, NOTES: 2, TASKS: 3 } as const;

function block(type: BlockLike["type"], text: string, checked = false): BlockLike {
  return { type, text, checked };
}


test("every notes key starts with the notes domain segment", () => {
  const keys = [
    queryKeys.notesList(),
    queryKeys.note("n1"),
    queryKeys.notebooks(),
    queryKeys.noteSearch("roadmap", SearchFacet.ALL),
    queryKeys.noteShares("n1"),
    queryKeys.noteComments("n1"),
    queryKeys.noteActivity("n1"),
    queryKeys.sharedWithMe(),
  ];
  for (const key of keys) {
    assert.equal(key[0], "notes", `key ${JSON.stringify(key)} is not domain-prefixed`);
  }
  assert.equal(queryKeys.notes[0], "notes");
});

test("an id is part of the key it identifies", () => {
  assert.notDeepEqual(queryKeys.note("a"), queryKeys.note("b"));
  assert.notDeepEqual(queryKeys.noteComments("a"), queryKeys.noteComments("b"));
  assert.notDeepEqual(queryKeys.noteActivity("a"), queryKeys.noteActivity("b"));
});

test("a note id and a notebook id never share a shares key", () => {
  assert.notDeepEqual(queryKeys.noteShares("x", "note"), queryKeys.noteShares("x", "notebook"));
});

test("the search palette keys by the trimmed query and the facet", () => {
  assert.deepEqual(queryKeys.noteSearch("  Roadmap ", SearchFacet.ALL), queryKeys.noteSearch("roadmap", SearchFacet.ALL));
  assert.notDeepEqual(
    queryKeys.noteSearch("roadmap", SearchFacet.NOTES),
    queryKeys.noteSearch("roadmap", SearchFacet.TASKS),
  );
});


test("equivalent note filters normalize to one key", () => {
  const spelled = normalizeNoteFilters({
    notebookId: "nb1",
    starredOnly: false,
    includeArchived: false,
    sharedOnly: false,
    sort: NoteSort.UNSPECIFIED,
    pageSize: 0,
  });
  assert.deepEqual(spelled, normalizeNoteFilters({ notebookId: "nb1" }));
  assert.deepEqual(normalizeNoteFilters(), normalizeNoteFilters({}));
  assert.deepEqual(queryKeys.notesList({ notebookId: "nb1" }), queryKeys.notesList(spelled));
});

test("a filter that changes the result changes the key", () => {
  assert.notDeepEqual(normalizeNoteFilters({ notebookId: "a" }), normalizeNoteFilters({ notebookId: "b" }));
  assert.notDeepEqual(normalizeNoteFilters({}), normalizeNoteFilters({ starredOnly: true }));
  assert.notDeepEqual(normalizeNoteFilters({}), normalizeNoteFilters({ includeArchived: true }));
  assert.notDeepEqual(normalizeNoteFilters({}), normalizeNoteFilters({ sharedOnly: true }));
  assert.notDeepEqual(normalizeNoteFilters({}), normalizeNoteFilters({ sort: NoteSort.TITLE }));
  assert.notDeepEqual(normalizeNoteFilters({}), normalizeNoteFilters({ pageSize: 50 }));
  assert.notDeepEqual(
    queryKeys.notesList({ starredOnly: true }),
    queryKeys.notesList({ starredOnly: false }),
  );
});

test("the sort default and an explicit default are one cache entry", () => {
  assert.deepEqual(
    queryKeys.notesList({ sort: NoteSort.UNSPECIFIED }),
    queryKeys.notesList({}),
  );
});


test("blocksToPlainText joins the text blocks, one per line", () => {
  const text = blocksToPlainText([
    block(BlockType.HEADING, "Roadmap"),
    block(BlockType.PARAGRAPH, "Ship the editor."),
    block(BlockType.TODO, "Write the tests", true),
  ]);
  assert.equal(text, "Roadmap\nShip the editor.\nWrite the tests");
});

test("blocksToPlainText skips what carries no text", () => {
  const text = blocksToPlainText([
    block(BlockType.PARAGRAPH, "before"),
    block(BlockType.DIVIDER, ""),
    { type: BlockType.IMAGE, text: "" },
    block(BlockType.PARAGRAPH, "   "),
    block(BlockType.PARAGRAPH, "  after  "),
  ]);
  assert.equal(text, "before\nafter", "blank lines would show up in the list row's preview");
});

test("blocksToPlainText handles an empty or missing block array", () => {
  assert.equal(blocksToPlainText([]), "");
  assert.equal(blocksToPlainText(undefined), "");
});

test("countTasks counts only TODO blocks", () => {
  const counts = countTasks([
    block(BlockType.TODO, "a", true),
    block(BlockType.TODO, "b"),
    block(BlockType.TODO, "c", true),
    block(BlockType.PARAGRAPH, "not a task"),
    block(BlockType.BULLET, "also not a task", true),
  ]);
  assert.deepEqual(counts, { total: 3, done: 2 });
});

test("countTasks on a note with no tasks", () => {
  assert.deepEqual(countTasks([block(BlockType.PARAGRAPH, "prose")]), { total: 0, done: 0 });
  assert.deepEqual(countTasks([]), { total: 0, done: 0 });
  assert.deepEqual(countTasks(undefined), { total: 0, done: 0 });
});

test("a checkbox toggle moves only the done count", () => {
  const blocks = [block(BlockType.TODO, "a"), block(BlockType.TODO, "b")];
  const before = countTasks(blocks);
  const after = countTasks([{ ...blocks[0]!, checked: true }, blocks[1]!]);
  assert.deepEqual(before, { total: 2, done: 0 });
  assert.deepEqual(after, { total: 2, done: 1 });
});
