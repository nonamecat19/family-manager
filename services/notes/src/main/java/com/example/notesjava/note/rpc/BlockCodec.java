package com.example.notesjava.note.rpc;

import com.google.protobuf.InvalidProtocolBufferException;
import com.google.protobuf.util.JsonFormat;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.BlockType;

import java.util.ArrayList;
import java.util.List;

public final class BlockCodec {

    private static final JsonFormat.Parser PARSER = JsonFormat.parser().ignoringUnknownFields();
    private static final JsonFormat.Printer PRINTER = JsonFormat.printer()
            .omittingInsignificantWhitespace()
            .alwaysPrintFieldsWithNoPresence();

    private BlockCodec() {
    }

    public static String encode(List<Block> blocks) {
        List<String> parts = new ArrayList<>(blocks.size());
        for (Block block : blocks) {
            try {
                parts.add(PRINTER.print(block));
            } catch (InvalidProtocolBufferException ex) {
                throw new IllegalStateException("blocks cannot be written as JSON", ex);
            }
        }
        return "[" + String.join(",", parts) + "]";
    }

    public static List<Block> decode(String json, String fallbackContent) {
        if (json == null || json.isBlank()) {
            return fromPlainText(fallbackContent);
        }

        List<Block> blocks = new ArrayList<>();
        for (String object : splitObjects(json)) {
            Block.Builder builder = Block.newBuilder();
            try {
                PARSER.merge(object, builder);
            } catch (InvalidProtocolBufferException ex) {
                return fromPlainText(fallbackContent);
            }
            blocks.add(builder.build());
        }
        return blocks.isEmpty() ? fromPlainText(fallbackContent) : blocks;
    }

    public static List<Block> fromPlainText(String content) {
        if (content == null || content.isBlank()) {
            return List.of();
        }
        List<Block> blocks = new ArrayList<>();
        int index = 0;
        for (String line : content.split("\n", -1)) {
            if (line.isBlank()) {
                continue;
            }
            blocks.add(Block.newBuilder()
                    .setId("b" + (++index))
                    .setType(BlockType.BLOCK_TYPE_PARAGRAPH)
                    .setText(line)
                    .build());
        }
        return blocks;
    }

    public static String toPlainText(List<Block> blocks) {
        StringBuilder sb = new StringBuilder();
        for (Block block : blocks) {
            if (block.getText().isBlank()) {
                continue;
            }
            if (!sb.isEmpty()) {
                sb.append('\n');
            }
            sb.append(block.getText());
        }
        return sb.toString();
    }

    public static String preview(List<Block> blocks) {
        String text = toPlainText(blocks).replace('\n', ' ').trim();
        return text.length() <= 120 ? text : text.substring(0, 119) + "…";
    }

    public static int taskTotal(List<Block> blocks) {
        return (int) blocks.stream().filter(b -> b.getType() == BlockType.BLOCK_TYPE_TODO).count();
    }

    public static int taskDone(List<Block> blocks) {
        return (int) blocks.stream()
                .filter(b -> b.getType() == BlockType.BLOCK_TYPE_TODO && b.getChecked())
                .count();
    }

    private static List<String> splitObjects(String json) {
        List<String> objects = new ArrayList<>();
        int depth = 0;
        int start = -1;
        boolean inString = false;
        boolean escaped = false;

        for (int i = 0; i < json.length(); i++) {
            char c = json.charAt(i);
            if (inString) {
                if (escaped) {
                    escaped = false;
                } else if (c == '\\') {
                    escaped = true;
                } else if (c == '"') {
                    inString = false;
                }
                continue;
            }
            switch (c) {
                case '"' -> inString = true;
                case '{' -> {
                    if (depth == 0) {
                        start = i;
                    }
                    depth++;
                }
                case '}' -> {
                    depth--;
                    if (depth == 0 && start >= 0) {
                        objects.add(json.substring(start, i + 1));
                        start = -1;
                    }
                }
                default -> {
                }
            }
        }
        return objects;
    }
}
