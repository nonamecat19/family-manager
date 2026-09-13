package com.example.notesjava.note.service;

import com.example.notesjava.common.error.InvalidRequestException;
import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.common.security.CallerContext;
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

import java.util.UUID;

@Service
@Transactional(readOnly = true)
public class NoteService {

    private final NoteRepository noteRepository;
    private final GroupRepository groupRepository;
    private final CallerContext callerContext;

    public NoteService(NoteRepository noteRepository, GroupRepository groupRepository, CallerContext callerContext) {
        this.noteRepository = noteRepository;
        this.groupRepository = groupRepository;
        this.callerContext = callerContext;
    }

    public Page<NoteResponse> list(Long groupId, NoteStatus status, Pageable pageable) {
        return noteRepository.search(familyId(), groupId, status, pageable).map(NoteResponse::from);
    }

    public NoteResponse getById(Long id) {
        return NoteResponse.from(noteRepository.findWithRelationsByIdAndFamilyId(id, familyId())
                .orElseThrow(() -> new ResourceNotFoundException("Note", id)));
    }

    @Transactional
    public NoteResponse create(CreateNoteRequest request) {
        UUID familyId = familyId();
        Note note = Note.builder()
                .familyId(familyId)
                .title(request.title())
                .content(request.content())
                .status(request.status())
                .priority(request.priority())
                .parent(resolveParent(familyId, request.parentId(), null))
                .group(resolveGroup(familyId, request.groupId()))
                .build();
        return NoteResponse.from(noteRepository.save(note));
    }

    @Transactional
    public NoteResponse update(Long id, UpdateNoteRequest request) {
        UUID familyId = familyId();
        Note note = require(familyId, id);
        note.edit(request.title(), request.content(), request.status(), request.priority());
        note.reparent(resolveParent(familyId, request.parentId(), id));
        note.moveTo(resolveGroup(familyId, request.groupId()));
        return NoteResponse.from(noteRepository.saveAndFlush(note));
    }

    @Transactional
    public void delete(Long id) {
        noteRepository.delete(require(familyId(), id));
    }

    private UUID familyId() {
        return callerContext.requireFamilyId();
    }

    private Note require(UUID familyId, Long id) {
        return noteRepository.findByIdAndFamilyId(id, familyId)
                .orElseThrow(() -> new ResourceNotFoundException("Note", id));
    }

    Note resolveParent(UUID familyId, Long parentId, Long selfId) {
        if (parentId == null) {
            return null;
        }
        if (parentId.equals(selfId)) {
            throw new InvalidRequestException("A note cannot be its own parent.");
        }
        Note parent = noteRepository.findByIdAndFamilyId(parentId, familyId)
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

    private Group resolveGroup(UUID familyId, Long groupId) {
        if (groupId == null) {
            return null;
        }
        return groupRepository.findByIdAndFamilyId(groupId, familyId)
                .orElseThrow(() -> new ResourceNotFoundException("Group", groupId));
    }
}
