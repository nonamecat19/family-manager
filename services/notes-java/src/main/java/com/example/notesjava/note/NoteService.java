package com.example.notesjava.note;

import com.example.notesjava.common.error.InvalidRequestException;
import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.group.Group;
import com.example.notesjava.group.GroupRepository;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * Returns DTOs rather than entities. {@code open-in-view} is off, so a {@code Note} mapped
 * after the method returns would be detached and its lazy {@code parent}/{@code group} would
 * blow up in the controller.
 */
@Service
@Transactional(readOnly = true)
public class NoteService {

    private final NoteRepository noteRepository;
    private final GroupRepository groupRepository;

    public NoteService(NoteRepository noteRepository, GroupRepository groupRepository) {
        this.noteRepository = noteRepository;
        this.groupRepository = groupRepository;
    }

    public Page<NoteResponse> list(Long groupId, NoteStatus status, Pageable pageable) {
        return noteRepository.search(groupId, status, pageable).map(NoteResponse::from);
    }

    public NoteResponse getById(Long id) {
        return NoteResponse.from(noteRepository.findWithRelationsById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Note", id)));
    }

    @Transactional
    public NoteResponse create(CreateNoteRequest request) {
        Note note = Note.builder()
                .title(request.title())
                .content(request.content())
                .status(request.status())
                .priority(request.priority())
                .parent(resolveParent(request.parentId(), null))
                .group(resolveGroup(request.groupId()))
                .build();
        return NoteResponse.from(noteRepository.save(note));
    }

    @Transactional
    public NoteResponse update(Long id, UpdateNoteRequest request) {
        Note note = require(id);
        note.edit(request.title(), request.content(), request.status(), request.priority());
        note.reparent(resolveParent(request.parentId(), id));
        note.moveTo(resolveGroup(request.groupId()));
        return NoteResponse.from(noteRepository.saveAndFlush(note));
    }

    /** Cascades to the subtree: deleting a note deletes everything filed under it. */
    @Transactional
    public void delete(Long id) {
        noteRepository.delete(require(id));
    }

    private Note require(Long id) {
        return noteRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Note", id));
    }

    /**
     * Walks the candidate parent's ancestry looking for the note being edited. Without this a
     * client could close a loop in the tree, and every later traversal of it would not
     * terminate.
     */
    Note resolveParent(Long parentId, Long selfId) {
        if (parentId == null) {
            return null;
        }
        if (parentId.equals(selfId)) {
            throw new InvalidRequestException("A note cannot be its own parent.");
        }
        Note parent = noteRepository.findById(parentId)
                .orElseThrow(() -> new ResourceNotFoundException("Note", parentId));
        if (selfId != null && isDescendantOf(parent, selfId)) {
            throw new InvalidRequestException(
                    "Note " + parentId + " is a descendant of note " + selfId + "; that would make a cycle.");
        }
        return parent;
    }

    private boolean isDescendantOf(Note node, Long ancestorId) {
        for (Note current = node; current != null; current = current.getParent()) {
            if (ancestorId.equals(current.getId())) {
                return true;
            }
        }
        return false;
    }

    private Group resolveGroup(Long groupId) {
        if (groupId == null) {
            return null;
        }
        return groupRepository.findById(groupId)
                .orElseThrow(() -> new ResourceNotFoundException("Group", groupId));
    }
}
