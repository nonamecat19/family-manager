package com.example.notesjava.note.rpc;

import com.nnc.familymanager.notes.v1.ActivityKind;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.BlockType;

import java.util.List;

record EditKind(ActivityKind kind, String detail) {

    private static final EditKind EDITED = new EditKind(ActivityKind.ACTIVITY_KIND_EDITED, "");

    static EditKind classify(String oldTitle, List<Block> oldBlocks, String newTitle, List<Block> newBlocks) {
        if (!oldTitle.equals(newTitle) || oldBlocks.size() != newBlocks.size()) {
            return EDITED;
        }
        int toggled = -1;
        for (int i = 0; i < newBlocks.size(); i++) {
            Block before = oldBlocks.get(i);
            Block after = newBlocks.get(i);
            if (!sameContent(before, after)) {
                return EDITED;
            }
            if (before.getChecked() != after.getChecked()) {
                if (toggled >= 0) {
                    return EDITED;
                }
                toggled = i;
            }
        }
        if (toggled < 0 || newBlocks.get(toggled).getType() != BlockType.BLOCK_TYPE_TODO) {
            return EDITED;
        }
        Block block = newBlocks.get(toggled);
        return new EditKind(
                block.getChecked() ? ActivityKind.ACTIVITY_KIND_TASK_CHECKED : ActivityKind.ACTIVITY_KIND_TASK_UNCHECKED,
                BlockCodec.preview(block.getText()));
    }

    private static boolean sameContent(Block before, Block after) {
        return before.getId().equals(after.getId())
                && before.getType() == after.getType()
                && before.getText().equals(after.getText())
                && before.getLevel() == after.getLevel()
                && before.getImageUrl().equals(after.getImageUrl())
                && before.getLanguage().equals(after.getLanguage());
    }
}
