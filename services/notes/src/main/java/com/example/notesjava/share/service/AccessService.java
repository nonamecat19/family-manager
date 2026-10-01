package com.example.notesjava.share.service;

import com.example.notesjava.group.domain.Group;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.share.domain.Share;
import com.example.notesjava.share.repository.ShareRepository;
import org.springframework.stereotype.Service;

import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

@Service
public class AccessService {

    private final ShareRepository shares;

    public AccessService(ShareRepository shares) {
        this.shares = shares;
    }

    public NoteAccess note(UUID familyId, UUID userId, Note note) {
        return notes(familyId, userId, List.of(note)).get(note.getId());
    }

    public Map<Long, NoteAccess> notes(UUID familyId, UUID userId, Collection<Note> notes) {
        Map<Long, NoteAccess> access = new HashMap<>();
        if (notes.isEmpty()) {
            return access;
        }

        Set<Long> noteIds = new HashSet<>();
        Set<Long> notebookIds = new HashSet<>();
        for (Note note : notes) {
            noteIds.add(note.getId());
            if (note.getGroup() != null) {
                notebookIds.add(note.getGroup().getId());
            }
        }

        Map<Long, List<Share>> direct = shares.findByFamilyIdAndNoteIdIn(familyId, noteIds).stream()
                .sorted(Share.ORDER)
                .collect(Collectors.groupingBy(Share::getNoteId));
        Map<Long, List<Share>> byNotebook = notebookIds.isEmpty()
                ? Map.of()
                : shares.findByFamilyIdAndNotebookIdIn(familyId, notebookIds).stream()
                        .collect(Collectors.groupingBy(Share::getNotebookId));

        for (Note note : notes) {
            List<Share> notebookShares = note.getGroup() == null
                    ? List.of()
                    : byNotebook.getOrDefault(note.getGroup().getId(), List.of());
            access.put(note.getId(), AccessPolicy.note(
                    note, userId, direct.getOrDefault(note.getId(), List.of()), notebookShares));
        }
        return access;
    }

    public NotebookAccess notebook(UUID familyId, UUID userId, Group group) {
        return AccessPolicy.notebook(group, userId,
                shares.findByFamilyIdAndNotebookIdOrderByCreatedAtAscIdAsc(familyId, group.getId()));
    }
}
