package com.example.notesjava.note.rpc;

import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.BlockType;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

class BlockCodecTest {

    private static Block block(BlockType type, String text, boolean checked) {
        return Block.newBuilder().setId("b1").setType(type).setText(text).setChecked(checked).build();
    }

    @Test
    void blocksSurviveARoundTrip() {
        List<Block> blocks = List.of(
                block(BlockType.BLOCK_TYPE_HEADING, "Shopping", false),
                block(BlockType.BLOCK_TYPE_TODO, "milk", true),
                block(BlockType.BLOCK_TYPE_TODO, "bread", false));

        List<Block> decoded = BlockCodec.decode(BlockCodec.encode(blocks), null);

        assertThat(decoded).hasSize(3);
        assertThat(decoded.get(0).getType()).isEqualTo(BlockType.BLOCK_TYPE_HEADING);
        assertThat(decoded.get(1).getChecked()).isTrue();
        assertThat(decoded.get(2).getText()).isEqualTo("bread");
    }

    @Test
    void textWithBracesAndQuotesSurvives() {
        List<Block> blocks = List.of(block(BlockType.BLOCK_TYPE_PARAGRAPH, "a {\"tricky\"} } value", false));

        List<Block> decoded = BlockCodec.decode(BlockCodec.encode(blocks), null);

        assertThat(decoded).hasSize(1);
        assertThat(decoded.getFirst().getText()).isEqualTo("a {\"tricky\"} } value");
    }

    @Test
    void aNoteWrittenBeforeBlocksExistedReadsAsParagraphs() {
        List<Block> decoded = BlockCodec.decode(null, "first line\n\nsecond line");

        assertThat(decoded).hasSize(2);
        assertThat(decoded.getFirst().getType()).isEqualTo(BlockType.BLOCK_TYPE_PARAGRAPH);
        assertThat(decoded.getFirst().getText()).isEqualTo("first line");
        assertThat(decoded.get(1).getText()).isEqualTo("second line");
    }

    @Test
    void malformedStoredBlocksFallBackToTheLegacyContent() {
        List<Block> decoded = BlockCodec.decode("{not json at all", "legacy body");

        assertThat(decoded).hasSize(1);
        assertThat(decoded.getFirst().getText()).isEqualTo("legacy body");
    }

    @Test
    void anEmptyNoteHasNoBlocks() {
        assertThat(BlockCodec.decode(null, null)).isEmpty();
        assertThat(BlockCodec.decode("[]", "")).isEmpty();
    }

    @Test
    void taskCountsAndPreviewAreDerived() {
        List<Block> blocks = List.of(
                block(BlockType.BLOCK_TYPE_PARAGRAPH, "Plan", false),
                block(BlockType.BLOCK_TYPE_TODO, "one", true),
                block(BlockType.BLOCK_TYPE_TODO, "two", false));

        assertThat(BlockCodec.taskTotal(blocks)).isEqualTo(2);
        assertThat(BlockCodec.taskDone(blocks)).isEqualTo(1);
        assertThat(BlockCodec.preview(blocks)).isEqualTo("Plan one two");
    }

    @Test
    void previewIsTruncated() {
        List<Block> blocks = List.of(block(BlockType.BLOCK_TYPE_PARAGRAPH, "x".repeat(400), false));

        assertThat(BlockCodec.preview(blocks)).hasSize(120).endsWith("…");
    }
}
