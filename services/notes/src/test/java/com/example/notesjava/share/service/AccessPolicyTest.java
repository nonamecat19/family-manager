package com.example.notesjava.share.service;

import com.example.notesjava.group.domain.Group;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.share.domain.Share;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;

class AccessPolicyTest {

    private static final UUID FAMILY = UUID.fromString("22222222-2222-2222-2222-222222222222");
    private static final UUID OWNER = UUID.fromString("11111111-1111-1111-1111-111111111111");
    private static final UUID MEMBER = UUID.fromString("33333333-3333-3333-3333-333333333333");
    private static final UUID STRANGER = UUID.fromString("55555555-5555-5555-5555-555555555555");

    private static Note note(UUID owner, Group group) {
        return Note.builder().familyId(FAMILY).title("t").ownerUserId(owner).group(group).build();
    }

    private static Group notebook(UUID owner) {
        return Group.builder().familyId(FAMILY).title("book").ownerUserId(owner).build();
    }

    private static Share toMember(short permission) {
        return Share.forNote(FAMILY, 1L, MEMBER, permission, OWNER);
    }

    private static Share toFamily(short permission) {
        return Share.forNotebook(FAMILY, 1L, null, permission, OWNER);
    }

    @Test
    void theOwnerSeesEditsAndManagesAnUnsharedNote() {
        NoteAccess access = AccessPolicy.note(note(OWNER, null), OWNER, List.of(), List.of());

        assertThat(access).isEqualTo(new NoteAccess(true, true, true, false, List.of()));
    }

    @Test
    void anUnsharedNoteIsHiddenFromEveryoneElse() {
        assertThat(AccessPolicy.note(note(OWNER, null), MEMBER, List.of(), List.of())).isEqualTo(NoteAccess.HIDDEN);
        assertThat(AccessPolicy.note(note(OWNER, null), null, List.of(), List.of())).isEqualTo(NoteAccess.HIDDEN);
    }

    @Test
    void aMemberShareReachesOnlyThatMemberWithItsPermission() {
        List<Share> view = List.of(toMember(Share.VIEW));
        List<Share> edit = List.of(toMember(Share.EDIT));

        NoteAccess viewing = AccessPolicy.note(note(OWNER, null), MEMBER, view, List.of());
        assertThat(viewing.visible()).isTrue();
        assertThat(viewing.canEdit()).isFalse();
        assertThat(viewing.canManage()).isFalse();
        assertThat(viewing.shared()).isTrue();

        assertThat(AccessPolicy.note(note(OWNER, null), MEMBER, edit, List.of()).canEdit()).isTrue();
        assertThat(AccessPolicy.note(note(OWNER, null), STRANGER, edit, List.of()).visible()).isFalse();
    }

    @Test
    void aNotebookShareCountsOnlyWhenTheNotebookBelongsToTheNotesOwner() {
        List<Share> family = List.of(toFamily(Share.EDIT));

        NoteAccess ownersNote = AccessPolicy.note(note(OWNER, notebook(OWNER)), STRANGER, List.of(), family);
        assertThat(ownersNote.visible()).isTrue();
        assertThat(ownersNote.canEdit()).isTrue();
        assertThat(ownersNote.shares()).isEmpty();

        NoteAccess parkedNote = AccessPolicy.note(note(MEMBER, notebook(OWNER)), STRANGER, List.of(), family);
        assertThat(parkedNote).isEqualTo(NoteAccess.HIDDEN);
        assertThat(AccessPolicy.note(note(MEMBER, notebook(OWNER)), MEMBER, List.of(), family).shared()).isFalse();
    }

    @Test
    void aNoteWithNoOwnerBelongsToTheWholeFamily() {
        NoteAccess access = AccessPolicy.note(note(null, null), STRANGER, List.of(), List.of());

        assertThat(access.visible()).isTrue();
        assertThat(access.canEdit()).isTrue();
        assertThat(access.canManage()).isTrue();
    }

    @Test
    void notebookAccessFollowsOwnershipAndShares() {
        assertThat(AccessPolicy.notebook(notebook(OWNER), OWNER, List.of()))
                .isEqualTo(new NotebookAccess(true, true, true));
        assertThat(AccessPolicy.notebook(notebook(null), STRANGER, List.of()))
                .isEqualTo(new NotebookAccess(true, true, true));
        assertThat(AccessPolicy.notebook(notebook(OWNER), STRANGER, List.of())).isEqualTo(NotebookAccess.HIDDEN);
        assertThat(AccessPolicy.notebook(notebook(OWNER), STRANGER, List.of(toFamily(Share.VIEW))))
                .isEqualTo(new NotebookAccess(true, false, false));
        assertThat(AccessPolicy.notebook(notebook(OWNER), STRANGER, List.of(toFamily(Share.EDIT))))
                .isEqualTo(new NotebookAccess(true, true, false));
    }
}
