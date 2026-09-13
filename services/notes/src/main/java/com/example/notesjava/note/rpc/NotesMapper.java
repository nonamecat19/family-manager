package com.example.notesjava.note.rpc;

import com.example.notesjava.group.domain.Group;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NoteActivity;
import com.example.notesjava.note.domain.NoteComment;
import com.google.protobuf.Timestamp;
import com.nnc.familymanager.notes.v1.Activity;
import com.nnc.familymanager.notes.v1.ActivityKind;
import com.nnc.familymanager.notes.v1.Block;
import com.nnc.familymanager.notes.v1.Comment;
import com.nnc.familymanager.notes.v1.Notebook;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

public final class NotesMapper {

    private NotesMapper() {
    }

    public static com.nnc.familymanager.notes.v1.Note toProto(Note note) {
        List<Block> blocks = BlockCodec.decode(note.getBlocks(), note.getContent());
        Group group = note.getGroup();

        com.nnc.familymanager.notes.v1.Note.Builder builder = com.nnc.familymanager.notes.v1.Note.newBuilder()
                .setId(String.valueOf(note.getId()))
                .setFamilyId(text(note.getFamilyId()))
                .setOwnerUserId(text(note.getOwnerUserId()))
                .setTitle(note.getTitle())
                .addAllBlocks(blocks)
                .setStarred(note.isStarred())
                .setArchived(note.isArchived())
                .setVersion(note.getVersion() == null ? 0 : note.getVersion())
                .setPreview(BlockCodec.preview(blocks))
                .setTaskTotal(BlockCodec.taskTotal(blocks))
                .setTaskDone(BlockCodec.taskDone(blocks))
                .setCanEdit(true);

        if (group != null) {
            builder.setNotebookId(String.valueOf(group.getId()));
        }
        if (note.getCreatedAt() != null) {
            builder.setCreatedAt(timestamp(note.getCreatedAt()));
        }
        if (note.getUpdatedAt() != null) {
            builder.setUpdatedAt(timestamp(note.getUpdatedAt()));
        }
        return builder.build();
    }

    public static Notebook toProto(Group group, long noteCount) {
        Notebook.Builder builder = Notebook.newBuilder()
                .setId(String.valueOf(group.getId()))
                .setFamilyId(text(group.getFamilyId()))
                .setName(group.getTitle())
                .setArchived(false)
                .setNoteCount((int) noteCount);

        if (group.getCreatedAt() != null) {
            builder.setCreatedAt(timestamp(group.getCreatedAt()));
        }
        if (group.getUpdatedAt() != null) {
            builder.setUpdatedAt(timestamp(group.getUpdatedAt()));
        }
        return builder.build();
    }

    public static Comment toProto(NoteComment comment) {
        Comment.Builder builder = Comment.newBuilder()
                .setId(String.valueOf(comment.getId()))
                .setNoteId(String.valueOf(comment.getNoteId()))
                .setAuthorUserId(text(comment.getAuthorUserId()))
                .setBody(comment.getBody())
                .setResolved(comment.isResolved());
        if (comment.getCreatedAt() != null) {
            builder.setCreatedAt(timestamp(comment.getCreatedAt()));
        }
        return builder.build();
    }

    public static Activity toProto(NoteActivity activity) {
        ActivityKind kind = ActivityKind.forNumber(activity.getKind());
        Activity.Builder builder = Activity.newBuilder()
                .setId(String.valueOf(activity.getId()))
                .setNoteId(String.valueOf(activity.getNoteId()))
                .setActorUserId(text(activity.getActorUserId()))
                .setKind(kind == null ? ActivityKind.ACTIVITY_KIND_UNSPECIFIED : kind)
                .setDetail(activity.getDetail());
        if (activity.getCreatedAt() != null) {
            builder.setCreatedAt(timestamp(activity.getCreatedAt()));
        }
        return builder.build();
    }

    public static Timestamp timestamp(Instant instant) {
        return Timestamp.newBuilder()
                .setSeconds(instant.getEpochSecond())
                .setNanos(instant.getNano())
                .build();
    }

    private static String text(UUID value) {
        return value == null ? "" : value.toString();
    }
}
