package com.example.notesjava.group;

import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.note.NoteRepository;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/**
 * Returns DTOs rather than entities. {@code open-in-view} is off, so anything mapped after the
 * method returns would be mapping a detached entity.
 */
@Service
@Transactional(readOnly = true)
public class GroupService {

    private final GroupRepository groupRepository;
    private final NoteRepository noteRepository;

    public GroupService(GroupRepository groupRepository, NoteRepository noteRepository) {
        this.groupRepository = groupRepository;
        this.noteRepository = noteRepository;
    }

    public Page<GroupResponse> list(Pageable pageable) {
        return groupRepository.findAll(pageable).map(GroupResponse::from);
    }

    public GroupResponse getById(Long id) {
        return GroupResponse.from(require(id));
    }

    @Transactional
    public GroupResponse create(CreateGroupRequest request) {
        Group group = Group.builder()
                .title(request.title())
                .color(request.color())
                .build();
        return GroupResponse.from(groupRepository.save(group));
    }

    @Transactional
    public GroupResponse update(Long id, UpdateGroupRequest request) {
        Group group = require(id);
        group.rename(request.title(), request.color());
        return GroupResponse.from(groupRepository.saveAndFlush(group));
    }

    /**
     * Deleting a group keeps its notes and unfiles them. The foreign key would do the same on
     * the database side, but the bulk update also evicts the now-stale notes from the
     * persistence context, so a later read in the same transaction cannot see a dangling group.
     */
    @Transactional
    public void delete(Long id) {
        Group group = require(id);
        noteRepository.clearGroup(group.getId());
        groupRepository.delete(group);
    }

    private Group require(Long id) {
        return groupRepository.findById(id)
                .orElseThrow(() -> new ResourceNotFoundException("Group", id));
    }
}
