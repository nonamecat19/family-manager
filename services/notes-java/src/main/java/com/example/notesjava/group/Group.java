package com.example.notesjava.group;

import com.example.notesjava.common.persistence.BaseEntity;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.EnumType;
import jakarta.persistence.Enumerated;
import jakarta.persistence.Table;
import lombok.AccessLevel;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

/**
 * A group is a flat label on notes, not their owner: the association is mapped on {@code Note}
 * only. Keeping a collection here would buy nothing and cost a lazy load on every read.
 */
@Entity
@Table(name = "groups")
@Getter
@Builder
@NoArgsConstructor(access = AccessLevel.PROTECTED)
@AllArgsConstructor(access = AccessLevel.PRIVATE)
public class Group extends BaseEntity {

    @Column(nullable = false)
    private String title;

    @Builder.Default
    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 32)
    private GroupColor color = GroupColor.DEFAULT;

    public static Group of(String title) {
        return Group.builder().title(title).build();
    }

    /**
     * The one mutator, so every state change goes through a named intent rather than a setter
     * per field.
     */
    public void rename(String title, GroupColor color) {
        this.title = title;
        this.color = color;
    }
}
