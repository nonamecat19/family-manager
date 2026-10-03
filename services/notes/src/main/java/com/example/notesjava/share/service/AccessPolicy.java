package com.example.notesjava.share.service;

import com.example.notesjava.group.domain.Group;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.share.domain.Share;

import java.util.List;
import java.util.UUID;

public final class AccessPolicy {

    private AccessPolicy() {
    }

    public static NoteAccess note(Note note, UUID userId, List<Share> direct, List<Share> notebookShares) {
        UUID owner = note.getOwnerUserId();
        if (owner == null) {
            return new NoteAccess(true, true, true, false, direct);
        }

        Group group = note.getGroup();
        List<Share> throughNotebook = group != null && owner.equals(group.getOwnerUserId())
                ? notebookShares
                : List.of();

        boolean mine = owner.equals(userId);
        boolean visible = mine || reaches(direct, userId, false) || reaches(throughNotebook, userId, false);
        if (!visible) {
            return NoteAccess.HIDDEN;
        }
        boolean canEdit = mine || reaches(direct, userId, true) || reaches(throughNotebook, userId, true);
        boolean shared = !direct.isEmpty() || !throughNotebook.isEmpty();
        return new NoteAccess(true, canEdit, mine, shared, direct);
    }

    public static NotebookAccess notebook(Group group, UUID userId, List<Share> shares) {
        UUID owner = group.getOwnerUserId();
        if (owner == null || owner.equals(userId)) {
            return new NotebookAccess(true, true, true);
        }
        if (!reaches(shares, userId, false)) {
            return NotebookAccess.HIDDEN;
        }
        return new NotebookAccess(true, reaches(shares, userId, true), false);
    }

    private static boolean reaches(List<Share> shares, UUID userId, boolean edit) {
        for (Share share : shares) {
            if (share.reaches(userId) && (!edit || share.allowsEdit())) {
                return true;
            }
        }
        return false;
    }
}
