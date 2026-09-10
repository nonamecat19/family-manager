package com.example.notesjava.note;

import com.example.notesjava.common.persistence.BaseEntity;
import com.example.notesjava.group.Group;
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

/**
 * Notes form a tree through {@code parent}. The children cascade, so deleting a note deletes
 * the subtree under it; the group does not, so deleting a group only unfiles its notes.
 */
@Entity
@Table(
        name = "notes",
        indexes = {
                @Index(name = "idx_notes_group_id", columnList = "group_id"),
                @Index(name = "idx_notes_parent_id", columnList = "parent_id"),
                @Index(name = "idx_notes_status", columnList = "status")
        }
)
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class Note extends BaseEntity {

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

    /**
     * Read-only from the Java side; {@code parent} is the owning end. Cascade, not orphan
     * removal: children are deleted with their parent, but moving one to another parent is a
     * foreign-key change, not a deletion.
     */
    @Builder.Default
    @OneToMany(mappedBy = "parent", cascade = CascadeType.ALL)
    private List<Note> childNotes = new ArrayList<>();

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "group_id")
    private Group group;

    public static Note of(String title, String content, Group group) {
        return Note.builder()
                .title(title)
                .content(content)
                .group(group)
                .build();
    }

    /** Unmodifiable: {@code parent} owns the association, so the collection is never written. */
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

    /**
     * Package-private: reparenting is only safe after the cycle check in
     * {@link NoteService#resolveParent}, which is the one caller.
     */
    void reparent(Note parent) {
        this.parent = parent;
    }
}
