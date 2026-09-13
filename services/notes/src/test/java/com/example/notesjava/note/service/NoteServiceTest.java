package com.example.notesjava.note.service;

import com.example.notesjava.common.error.InvalidRequestException;
import com.example.notesjava.common.error.ResourceNotFoundException;
import com.example.notesjava.common.persistence.BaseEntity;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.api.dto.CreateNoteRequest;
import com.example.notesjava.note.api.dto.NoteResponse;
import com.example.notesjava.note.api.dto.UpdateNoteRequest;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NotePriority;
import com.example.notesjava.note.domain.NoteStatus;
import com.example.notesjava.note.repository.NoteRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.test.util.ReflectionTestUtils;

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
class NoteServiceTest {

    @Mock
    private NoteRepository noteRepository;

    @Mock
    private GroupRepository groupRepository;

    @Mock
    private CallerContext callerContext;

    @InjectMocks
    private NoteService noteService;

    private static final UUID FAMILY = UUID.fromString("11111111-1111-1111-1111-111111111111");

    private Note existing;

    @BeforeEach
    void setUp() {
        existing = Note.of(FAMILY, "existing", "body", null);
        lenient().when(callerContext.requireFamilyId()).thenReturn(FAMILY);
        lenient().when(noteRepository.save(any(Note.class))).thenAnswer(call -> call.getArgument(0));
        lenient().when(noteRepository.saveAndFlush(any(Note.class))).thenAnswer(call -> call.getArgument(0));
    }

    @Test
    void createAppliesDefaultsWhenStatusAndPriorityAreAbsent() {
        NoteResponse created = noteService.create(
                new CreateNoteRequest("title", "content", null, null, null, null));

        assertThat(created.status()).isEqualTo(NoteStatus.DEFAULT);
        assertThat(created.priority()).isEqualTo(NotePriority.DEFAULT);
    }

    @Test
    void createRejectsAnUnknownGroup() {
        when(groupRepository.findByIdAndFamilyId(99L, FAMILY)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> noteService.create(
                new CreateNoteRequest("title", null, null, null, null, 99L)))
                .isInstanceOf(ResourceNotFoundException.class)
                .hasMessageContaining("Group 99");

        verify(noteRepository, never()).save(any());
    }

    @Test
    void createStampsTheCallersFamilyOnTheNote() {
        noteService.create(new CreateNoteRequest("title", null, null, null, null, null));

        org.mockito.ArgumentCaptor<Note> saved = org.mockito.ArgumentCaptor.forClass(Note.class);
        verify(noteRepository).save(saved.capture());
        assertThat(saved.getValue().getFamilyId()).isEqualTo(FAMILY);
    }

    @Test
    void getByIdRaisesNotFoundForAMissingNote() {
        when(noteRepository.findWithRelationsByIdAndFamilyId(7L, FAMILY)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> noteService.getById(7L))
                .isInstanceOf(ResourceNotFoundException.class)
                .hasMessageContaining("Note 7");
    }

    @Test
    void updateRefusesToMakeANoteItsOwnParent() {
        when(noteRepository.findByIdAndFamilyId(1L, FAMILY)).thenReturn(Optional.of(existing));

        assertThatThrownBy(() -> noteService.update(1L,
                new UpdateNoteRequest("t", null, NoteStatus.ACTIVE, NotePriority.NORMAL, 1L, null)))
                .isInstanceOf(InvalidRequestException.class)
                .hasMessageContaining("its own parent");
    }

    @Test
    void updateRefusesAParentThatSitsBelowTheNote() {
        Note root = noteWithId(1L, "root");
        Note child = noteWithId(2L, "child");
        child.reparent(root);
        when(noteRepository.findByIdAndFamilyId(1L, FAMILY)).thenReturn(Optional.of(root));
        when(noteRepository.findByIdAndFamilyId(2L, FAMILY)).thenReturn(Optional.of(child));

        assertThatThrownBy(() -> noteService.update(1L,
                new UpdateNoteRequest("t", null, NoteStatus.ACTIVE, NotePriority.NORMAL, 2L, null)))
                .isInstanceOf(InvalidRequestException.class)
                .hasMessageContaining("cycle");
    }

    @Test
    void updateWritesEveryMutableField() {
        when(noteRepository.findByIdAndFamilyId(1L, FAMILY)).thenReturn(Optional.of(existing));

        NoteResponse updated = noteService.update(1L,
                new UpdateNoteRequest("new title", "new body", NoteStatus.COMPLETED, NotePriority.HIGH, null, null));

        assertThat(updated.title()).isEqualTo("new title");
        assertThat(updated.content()).isEqualTo("new body");
        assertThat(updated.status()).isEqualTo(NoteStatus.COMPLETED);
        assertThat(updated.priority()).isEqualTo(NotePriority.HIGH);
        assertThat(updated.parentId()).isNull();
        assertThat(updated.groupId()).isNull();
    }

    @Test
    void reparentingUpwardsIsAllowedWhenThereIsNoCycle() {
        Note root = noteWithId(1L, "root");
        Note other = noteWithId(2L, "other");
        when(noteRepository.findByIdAndFamilyId(1L, FAMILY)).thenReturn(Optional.of(root));
        when(noteRepository.findByIdAndFamilyId(2L, FAMILY)).thenReturn(Optional.of(other));

        assertThat(noteService.update(1L,
                new UpdateNoteRequest("t", null, NoteStatus.ACTIVE, NotePriority.NORMAL, 2L, null)).parentId())
                .isEqualTo(2L);
    }

    @Test
    void deleteLoadsTheNoteFirstSoAMissingIdIsA404() {
        when(noteRepository.findByIdAndFamilyId(5L, FAMILY)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> noteService.delete(5L))
                .isInstanceOf(ResourceNotFoundException.class);

        verify(noteRepository, never()).delete(any());
    }

    private static Note noteWithId(Long id, String title) {
        Note note = Note.of(FAMILY, title, null, null);
        ReflectionTestUtils.setField(note, BaseEntity.class, "id", id, Long.class);
        return note;
    }
}
