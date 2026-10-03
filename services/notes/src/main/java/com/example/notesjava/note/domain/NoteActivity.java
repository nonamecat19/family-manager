package com.example.notesjava.note.domain;

import com.example.notesjava.common.persistence.BaseEntity;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.Table;
import lombok.AccessLevel;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.UUID;

@Entity
@Table(name = "note_activity")
@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
public class NoteActivity extends BaseEntity {

    @Column(name = "family_id", nullable = false, updatable = false)
    private UUID familyId;

    @Column(name = "note_id", nullable = false, updatable = false)
    private Long noteId;

    @Column(name = "actor_user_id", nullable = false, updatable = false)
    private UUID actorUserId;

    @Column(nullable = false, updatable = false)
    private short kind;

    @Column(nullable = false, updatable = false, columnDefinition = "TEXT")
    private String detail;

    public static NoteActivity of(UUID familyId, Long noteId, UUID actorUserId, int kind, String detail) {
        NoteActivity activity = new NoteActivity();
        activity.familyId = familyId;
        activity.noteId = noteId;
        activity.actorUserId = actorUserId;
        activity.kind = (short) kind;
        activity.detail = detail == null ? "" : detail;
        return activity;
    }
}
