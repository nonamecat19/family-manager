package com.example.notesjava.note.repository;

import com.example.notesjava.common.persistence.JpaAuditingConfig;
import com.example.notesjava.group.domain.Group;
import com.example.notesjava.group.repository.GroupRepository;
import com.example.notesjava.note.domain.Note;
import com.example.notesjava.note.domain.NotePriority;
import com.example.notesjava.note.domain.NoteStatus;
import com.example.notesjava.support.AbstractIntegrationTest;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.data.jpa.test.autoconfigure.DataJpaTest;
import org.springframework.context.annotation.Import;
import org.springframework.data.domain.PageRequest;

import static org.assertj.core.api.Assertions.assertThat;

@DataJpaTest
@Import(JpaAuditingConfig.class)
class NoteRepositoryTest extends AbstractIntegrationTest {

    @Autowired
    private NoteRepository noteRepository;

    @Autowired
    private GroupRepository groupRepository;

    @Test
    void auditingPopulatesTimestampsAndVersion() {
        Note saved = noteRepository.save(Note.of("first", "body", null));

        assertThat(saved.getId()).isNotNull();
        assertThat(saved.getCreatedAt()).isNotNull();
        assertThat(saved.getUpdatedAt()).isNotNull();
        assertThat(saved.getVersion()).isZero();
    }

    @Test
    void builderDefaultsSurviveThePersistRoundTrip() {
        Note saved = noteRepository.saveAndFlush(Note.builder().title("defaults").build());

        assertThat(saved.getStatus()).isEqualTo(NoteStatus.DEFAULT);
        assertThat(saved.getPriority()).isEqualTo(NotePriority.DEFAULT);
        assertThat(saved.getChildNotes()).isEmpty();
    }

    @Test
    void searchFiltersByGroupAndStatus() {
        Group work = groupRepository.save(Group.of("work"));
        noteRepository.save(Note.of("in group", null, work));
        noteRepository.save(Note.of("ungrouped", null, null));

        assertThat(noteRepository.search(work.getId(), null, PageRequest.of(0, 10)))
                .extracting(Note::getTitle)
                .containsExactly("in group");
        assertThat(noteRepository.search(null, NoteStatus.ACTIVE, PageRequest.of(0, 10)))
                .hasSize(2);
        assertThat(noteRepository.search(null, NoteStatus.COMPLETED, PageRequest.of(0, 10)))
                .isEmpty();
    }

    @Test
    void searchWithNoFiltersReturnsEverything() {
        noteRepository.save(Note.of("a", null, null));
        noteRepository.save(Note.of("b", null, null));

        assertThat(noteRepository.search(null, null, PageRequest.of(0, 10))).hasSize(2);
    }

    @Test
    void findWithRelationsLoadsParentAndGroupEagerly() {
        Group work = groupRepository.save(Group.of("work"));
        Note parent = noteRepository.save(Note.of("parent", null, work));
        Note child = noteRepository.saveAndFlush(
                Note.builder().title("child").parent(parent).group(work).build());

        Note loaded = noteRepository.findWithRelationsById(child.getId()).orElseThrow();

        assertThat(org.hibernate.Hibernate.isInitialized(loaded.getParent())).isTrue();
        assertThat(org.hibernate.Hibernate.isInitialized(loaded.getGroup())).isTrue();
    }

    @Test
    void clearGroupDetachesNotesWithoutDeletingThem() {
        Group work = groupRepository.save(Group.of("work"));
        Note note = noteRepository.save(Note.of("filed", null, work));

        int cleared = noteRepository.clearGroup(work.getId());

        assertThat(cleared).isEqualTo(1);
        assertThat(noteRepository.findById(note.getId()))
                .get()
                .extracting(Note::getGroup)
                .isNull();
    }
}
