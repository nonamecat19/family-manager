package com.example.notesjava.note.rpc;

import com.nnc.familymanager.notes.v1.ActivityKind;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.BlockType;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

class EditKindTest {

    private static Block todo(String id, String text, boolean checked) {
        return Block.newBuilder().setId(id).setType(BlockType.BLOCK_TYPE_TODO).setText(text).setChecked(checked).build();
    }

    private static Block paragraph(String id, String text) {
        return Block.newBuilder().setId(id).setType(BlockType.BLOCK_TYPE_PARAGRAPH).setText(text).build();
    }

    @Test
    void tickingOneTaskIsItsOwnLine() {
        EditKind edit = EditKind.classify(
                "Groceries", List.of(todo("a", "Buy milk", false), todo("b", "Eggs", false)),
                "Groceries", List.of(todo("a", "Buy milk", true), todo("b", "Eggs", false)));

        assertThat(edit.kind()).isEqualTo(ActivityKind.ACTIVITY_KIND_TASK_CHECKED);
        assertThat(edit.detail()).isEqualTo("Buy milk");
    }

    @Test
    void untickingOneTaskIsItsOwnLine() {
        EditKind edit = EditKind.classify(
                "Groceries", List.of(todo("a", "Buy milk", true)),
                "Groceries", List.of(todo("a", "Buy milk", false)));

        assertThat(edit.kind()).isEqualTo(ActivityKind.ACTIVITY_KIND_TASK_UNCHECKED);
    }

    @Test
    void tickingTwoTasksAtOnceIsAnEdit() {
        EditKind edit = EditKind.classify(
                "Groceries", List.of(todo("a", "Buy milk", false), todo("b", "Eggs", false)),
                "Groceries", List.of(todo("a", "Buy milk", true), todo("b", "Eggs", true)));

        assertThat(edit).isEqualTo(new EditKind(ActivityKind.ACTIVITY_KIND_EDITED, ""));
    }

    @Test
    void aTickAlongsideATextChangeIsAnEdit() {
        EditKind edit = EditKind.classify(
                "Groceries", List.of(todo("a", "Buy milk", false)),
                "Groceries", List.of(todo("a", "Buy oat milk", true)));

        assertThat(edit.kind()).isEqualTo(ActivityKind.ACTIVITY_KIND_EDITED);
    }

    @Test
    void renamingOrAddingABlockIsAnEdit() {
        List<Block> blocks = List.of(paragraph("a", "hello"));

        assertThat(EditKind.classify("Old", blocks, "New", blocks).kind())
                .isEqualTo(ActivityKind.ACTIVITY_KIND_EDITED);
        assertThat(EditKind.classify("Same", blocks, "Same", List.of(paragraph("a", "hello"), paragraph("b", "x"))).kind())
                .isEqualTo(ActivityKind.ACTIVITY_KIND_EDITED);
    }

    @Test
    void savingUnchangedContentIsAnEdit() {
        List<Block> blocks = List.of(paragraph("a", "hello"));

        assertThat(EditKind.classify("Same", blocks, "Same", blocks).kind())
                .isEqualTo(ActivityKind.ACTIVITY_KIND_EDITED);
    }
}
