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

import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;

@DataJpaTest
@Import(JpaAuditingConfig.class)
class NoteRepositoryTest extends AbstractIntegrationTest {

    @Autowired
    private NoteRepository noteRepository;

    @Autowired
    private GroupRepository groupRepository;

    private static final UUID FAMILY = UUID.fromString("11111111-1111-1111-1111-111111111111");
    private static final UUID OTHER_FAMILY = UUID.fromString("22222222-2222-2222-2222-222222222222");

    @Test
    void auditingPopulatesTimestampsAndVersion() {
        Note saved = noteRepository.save(Note.of(FAMILY, "first", "body", null));

        assertThat(saved.getId()).isNotNull();
        assertThat(saved.getCreatedAt()).isNotNull();
        assertThat(saved.getUpdatedAt()).isNotNull();
        assertThat(saved.getVersion()).isZero();
    }

    @Test
    void builderDefaultsSurviveThePersistRoundTrip() {
        Note saved = noteRepository.saveAndFlush(Note.builder().familyId(FAMILY).title("defaults").build());

        assertThat(saved.getStatus()).isEqualTo(NoteStatus.DEFAULT);
        assertThat(saved.getPriority()).isEqualTo(NotePriority.DEFAULT);
        assertThat(saved.getChildNotes()).isEmpty();
    }

    @Test
    void searchFiltersByGroupAndStatus() {
        Group work = groupRepository.save(Group.of(FAMILY, "work"));
        noteRepository.save(Note.of(FAMILY, "in group", null, work));
        noteRepository.save(Note.of(FAMILY, "ungrouped", null, null));

        assertThat(noteRepository.search(FAMILY, work.getId(), null, PageRequest.of(0, 10)))
                .extracting(Note::getTitle)
                .containsExactly("in group");
        assertThat(noteRepository.search(FAMILY, null, NoteStatus.ACTIVE, PageRequest.of(0, 10)))
                .hasSize(2);
        assertThat(noteRepository.search(FAMILY, null, NoteStatus.COMPLETED, PageRequest.of(0, 10)))
                .isEmpty();
    }

    @Test
    void searchWithNoFiltersReturnsEverything() {
        noteRepository.save(Note.of(FAMILY, "a", null, null));
        noteRepository.save(Note.of(FAMILY, "b", null, null));

        assertThat(noteRepository.search(FAMILY, null, null, PageRequest.of(0, 10))).hasSize(2);
    }

    @Test
    void findWithRelationsLoadsParentAndGroupEagerly() {
        Group work = groupRepository.save(Group.of(FAMILY, "work"));
        Note parent = noteRepository.save(Note.of(FAMILY, "parent", null, work));
        Note child = noteRepository.saveAndFlush(
                Note.builder().familyId(FAMILY).title("child").parent(parent).group(work).build());

        Note loaded = noteRepository.findWithRelationsByIdAndFamilyId(child.getId(), FAMILY).orElseThrow();

        assertThat(org.hibernate.Hibernate.isInitialized(loaded.getParent())).isTrue();
        assertThat(org.hibernate.Hibernate.isInitialized(loaded.getGroup())).isTrue();
    }

    @Test
    void searchNeverCrossesFamilyBoundaries() {
        noteRepository.save(Note.of(FAMILY, "ours", null, null));
        noteRepository.save(Note.of(OTHER_FAMILY, "theirs", null, null));

        assertThat(noteRepository.search(FAMILY, null, null, PageRequest.of(0, 10)))
                .extracting(Note::getTitle)
                .containsExactly("ours");
        assertThat(noteRepository.findByIdAndFamilyId(
                noteRepository.save(Note.of(OTHER_FAMILY, "hidden", null, null)).getId(), FAMILY))
                .isEmpty();
    }

    @Test
    void clearGroupLeavesAnotherFamilysNotesAlone() {
        Group ours = groupRepository.save(Group.of(FAMILY, "shared name"));
        Group theirs = groupRepository.save(Group.of(OTHER_FAMILY, "shared name"));
        noteRepository.save(Note.of(FAMILY, "ours", null, ours));
        Note theirNote = noteRepository.save(Note.of(OTHER_FAMILY, "theirs", null, theirs));

        assertThat(noteRepository.clearGroup(theirs.getId(), FAMILY)).isZero();
        assertThat(noteRepository.findById(theirNote.getId()))
                .get()
                .extracting(note -> note.getGroup().getId())
                .isEqualTo(theirs.getId());
    }

    @Test
    void clearGroupDetachesNotesWithoutDeletingThem() {
        Group work = groupRepository.save(Group.of(FAMILY, "work"));
        Note note = noteRepository.save(Note.of(FAMILY, "filed", null, work));

        int cleared = noteRepository.clearGroup(work.getId(), FAMILY);

        assertThat(cleared).isEqualTo(1);
        assertThat(noteRepository.findById(note.getId()))
                .get()
                .extracting(Note::getGroup)
                .isNull();
    }
}
