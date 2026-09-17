package com.example.notesjava.group.service;

import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.group.api.dto.CreateGroupRequest;
import com.example.notesjava.group.api.dto.GroupResponse;
import com.example.notesjava.group.api.dto.UpdateGroupRequest;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.repository.NoteRepository;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

@Service
@Transactional(readOnly = true)
public class GroupService {

    private final GroupRepository groupRepository;
    private final NoteRepository noteRepository;
    private final CallerContext callerContext;

    public GroupService(GroupRepository groupRepository, NoteRepository noteRepository, CallerContext callerContext) {
        this.groupRepository = groupRepository;
        this.noteRepository = noteRepository;
        this.callerContext = callerContext;
    }

    public Page<GroupResponse> list(Pageable pageable) {
        return groupRepository.findAllByFamilyId(familyId(), pageable).map(GroupResponse::from);
    }

    public GroupResponse getById(Long id) {
        return GroupResponse.from(require(familyId(), id));
    }

    @Transactional
    public GroupResponse create(CreateGroupRequest request) {
        Group group = Group.builder()
                .familyId(familyId())
                .title(request.title())
                .color(request.color())
                .build();
        return GroupResponse.from(groupRepository.save(group));
    }

    @Transactional
    public GroupResponse update(Long id, UpdateGroupRequest request) {
        Group group = require(familyId(), id);
        group.rename(request.title(), request.color());
        return GroupResponse.from(groupRepository.saveAndFlush(group));
    }

    @Transactional
    public void delete(Long id) {
        UUID familyId = familyId();
        Group group = require(familyId, id);
        noteRepository.clearGroup(group.getId(), familyId);
        groupRepository.delete(group);
    }

    private UUID familyId() {
        return callerContext.requireFamilyId();
    }

    private Group require(UUID familyId, Long id) {
        return groupRepository.findByIdAndFamilyId(id, familyId)
                .orElseThrow(() -> new ResourceNotFoundException("Group", id));
    }
}
