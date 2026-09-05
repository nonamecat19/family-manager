package com.example.notesjava.common.persistence;

import jakarta.persistence.Column;
import jakarta.persistence.EntityListeners;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.MappedSuperclass;
import jakarta.persistence.Version;
import lombok.Getter;
import org.springframework.data.annotation.CreatedDate;
import org.springframework.data.annotation.LastModifiedDate;
import org.springframework.data.jpa.domain.support.AuditingEntityListener;

import java.time.Instant;

@Getter
@MappedSuperclass
@EntityListeners(AuditingEntityListener.class)
public abstract class BaseEntity {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Version
    @Column(nullable = false)
    private Long version;

    @CreatedDate
    @Column(nullable = false, updatable = false)
    private Instant createdAt;

    @LastModifiedDate
    @Column(nullable = false)
    private Instant updatedAt;

    /**
     * Identity is the database id, so two unsaved instances are never equal. {@code getClass()}
     * rather than {@code instanceof} would break under Hibernate proxies, hence the unproxied
     * class comparison below.
     */
    @Override
    public final boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (!(o instanceof BaseEntity other) || !entityClass(this).equals(entityClass(other))) {
            return false;
        }
        return id != null && id.equals(other.id);
    }

    /**
     * Constant per type: the id is null before the insert and non-null after it, and a hash that
     * changed mid-transaction would lose the entity inside any HashSet holding it.
     */
    @Override
    public final int hashCode() {
        return entityClass(this).hashCode();
    }

    private static Class<?> entityClass(Object entity) {
        return org.hibernate.Hibernate.getClass(entity);
    }
}
