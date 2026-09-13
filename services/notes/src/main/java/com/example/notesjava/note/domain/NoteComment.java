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
@Table(name = "note_comments")
@Getter
@NoArgsConstructor(access = AccessLevel.PROTECTED)
public class NoteComment extends BaseEntity {

    @Column(name = "family_id", nullable = false, updatable = false)
    private UUID familyId;

    @Column(name = "note_id", nullable = false, updatable = false)
    private Long noteId;

    @Column(name = "author_user_id", nullable = false, updatable = false)
    private UUID authorUserId;

    @Column(nullable = false, updatable = false, columnDefinition = "TEXT")
    private String body;

    @Column(nullable = false)
    private boolean resolved;

    public static NoteComment of(UUID familyId, Long noteId, UUID authorUserId, String body) {
        NoteComment comment = new NoteComment();
        comment.familyId = familyId;
        comment.noteId = noteId;
        comment.authorUserId = authorUserId;
        comment.body = body;
        return comment;
    }

    public void resolve(boolean resolved) {
        this.resolved = resolved;
    }
}
