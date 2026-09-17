package com.example.notesjava.note.domain;

import com.example.notesjava.common.persistence.BaseEntity;
import com.example.notesjava.group.domain.Group;
import jakarta.persistence.CascadeType;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.FetchType;
import jakarta.persistence.Index;
import jakarta.persistence.JoinColumn;
import jakarta.persistence.ManyToOne;
import jakarta.persistence.OneToMany;
import jakarta.persistence.Table;
import lombok.AccessLevel;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.UUID;

@Entity
@Table(
        name = "notes",
        indexes = {
                @Index(name = "idx_notes_family_id", columnList = "family_id, updated_at"),
                @Index(name = "idx_notes_group_id", columnList = "family_id, group_id"),
                @Index(name = "idx_notes_parent_id", columnList = "parent_id"),
                @Index(name = "idx_notes_status", columnList = "family_id, status")
        }
)
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class Note extends BaseEntity {

    @Column(name = "family_id", nullable = false, updatable = false)
    private UUID familyId;

    @Column(nullable = false)
    private String title;

    @Column(columnDefinition = "TEXT")
    private String content;

    @Builder.Default
    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 32)
    private NoteStatus status = NoteStatus.DEFAULT;

    @Builder.Default
    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 32)
    private NotePriority priority = NotePriority.DEFAULT;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "parent_id")
    private Note parent;

    @Builder.Default
    @OneToMany(mappedBy = "parent", cascade = CascadeType.ALL)
    private List<Note> childNotes = new ArrayList<>();

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "group_id")
    private Group group;

    public static Note of(UUID familyId, String title, String content, Group group) {
        return Note.builder()
                .familyId(familyId)
                .title(title)
                .content(content)
                .group(group)
                .build();
    }

    public List<Note> getChildNotes() {
        return Collections.unmodifiableList(childNotes);
    }

    public void edit(String title, String content, NoteStatus status, NotePriority priority) {
        this.title = title;
        this.content = content;
        this.status = status;
        this.priority = priority;
    }

    public void moveTo(Group group) {
        this.group = group;
    }

    public void reparent(Note parent) {
        this.parent = parent;
    }
}
