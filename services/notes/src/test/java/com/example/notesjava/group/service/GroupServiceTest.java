package com.example.notesjava.group.service;

import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.group.api.dto.CreateGroupRequest;
import com.example.notesjava.group.api.dto.GroupResponse;
import com.example.notesjava.group.api.dto.UpdateGroupRequest;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.domain.GroupColor;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.repository.NoteRepository;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.Optional;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.lenient;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class GroupServiceTest {

    @Mock
    private GroupRepository groupRepository;

    @Mock
    private NoteRepository noteRepository;

    @Mock
    private CallerContext callerContext;

    @InjectMocks
    private GroupService groupService;

    private static final UUID FAMILY = UUID.fromString("11111111-1111-1111-1111-111111111111");

    @org.junit.jupiter.api.BeforeEach
    void stubCaller() {
        lenient().when(callerContext.requireFamilyId()).thenReturn(FAMILY);
    }

    @Test
    void createFallsBackToTheDefaultColour() {
        when(groupRepository.save(any(Group.class))).thenAnswer(call -> call.getArgument(0));

        GroupResponse created = groupService.create(new CreateGroupRequest("work", null));

        assertThat(created.color()).isEqualTo(GroupColor.DEFAULT);
        assertThat(created.title()).isEqualTo("work");
    }

    @Test
    void updateRewritesTitleAndColour() {
        Group group = Group.of(FAMILY, "old");
        when(groupRepository.findByIdAndFamilyId(1L, FAMILY)).thenReturn(Optional.of(group));
        when(groupRepository.saveAndFlush(any(Group.class))).thenAnswer(call -> call.getArgument(0));

        GroupResponse updated = groupService.update(1L, new UpdateGroupRequest("new", GroupColor.RED));

        assertThat(updated.title()).isEqualTo("new");
        assertThat(updated.color()).isEqualTo(GroupColor.RED);
    }

    @Test
    void deleteUnfilesTheNotesBeforeRemovingTheGroup() {
        Group group = Group.of(FAMILY, "work");
        when(groupRepository.findByIdAndFamilyId(1L, FAMILY)).thenReturn(Optional.of(group));

        groupService.delete(1L);

        verify(noteRepository).clearGroup(group.getId(), FAMILY);
        verify(groupRepository).delete(group);
    }

    @Test
    void deletingAMissingGroupIsA404AndTouchesNoNotes() {
        when(groupRepository.findByIdAndFamilyId(9L, FAMILY)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> groupService.delete(9L))
                .isInstanceOf(ResourceNotFoundException.class)
                .hasMessageContaining("Group 9");

        verify(noteRepository, never()).clearGroup(any(), any());
    }
}
