package com.example.notesjava.note.service;

import com.example.notesjava.common.error.InvalidRequestException;
import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.api.dto.CreateNoteRequest;
import com.example.notesjava.note.api.dto.NoteResponse;
import com.example.notesjava.note.api.dto.UpdateNoteRequest;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NoteStatus;
import com.example.notesjava.note.repository.NoteRepository;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

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

    @Transactional
    public void delete(Long id) {
        noteRepository.delete(require(id));
    }

    private Note require(Long id) {
        return noteRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Note", id));
    }

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
