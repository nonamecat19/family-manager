package com.example.notesjava.image;

import com.example.notesjava.common.connect.ConnectCode;
import com.example.notesjava.common.connect.ConnectException;
import com.example.notesjava.common.security.Caller;
import com.example.notesjava.common.security.CallerContext;
import com.example.notesjava.note.rpc.NotesRpcService;
import com.example.notesjava.support.AbstractIntegrationTest;
import com.google.protobuf.ByteString;
import com.nnc.familymanager.notes.v1.CreateNoteRequest;
import com.nnc.familymanager.notes.v1.CreateNoteResponse;
import com.nnc.familymanager.notes.v1.ShareNoteRequest;
import com.nnc.familymanager.notes.v1.SharePermission;
import com.nnc.familymanager.notes.v1.ShareSubject;
import com.nnc.familymanager.notes.v1.UploadNoteImageRequest;
import com.nnc.familymanager.notes.v1.UploadNoteImageResponse;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.testcontainers.containers.MinIOContainer;
import org.testcontainers.junit.jupiter.Container;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.UUID;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.catchThrowableOfType;
import static org.mockito.Mockito.when;

@SpringBootTest
class NoteImageUploadIT extends AbstractIntegrationTest {

    private static final UUID FAMILY = UUID.fromString("22222222-2222-2222-2222-222222222222");
    private static final String USER = "11111111-1111-1111-1111-111111111111";
    private static final String OTHER = "33333333-3333-3333-3333-333333333333";

    @Container
    static final MinIOContainer MINIO = new MinIOContainer("minio/minio:latest");

    @DynamicPropertySource
    static void storage(DynamicPropertyRegistry registry) {
        registry.add("notes.storage.provider", () -> "minio");
        registry.add("notes.storage.endpoint", MINIO::getS3URL);
        registry.add("notes.storage.access-key", MINIO::getUserName);
        registry.add("notes.storage.secret-key", MINIO::getPassword);
        registry.add("notes.storage.bucket", () -> "notes-it");
    }

    @Autowired
    private NotesRpcService rpc;

    @MockitoBean
    private CallerContext callers;

    @BeforeEach
    void caller() {
        actAs(USER);
    }

    private void actAs(String userId) {
        when(callers.require()).thenReturn(new Caller(userId, FAMILY, userId + "@example.test"));
        when(callers.requireFamilyId()).thenReturn(FAMILY);
    }

    private String createNote() {
        return ((CreateNoteResponse) rpc.procedure("CreateNote").invoke(
                CreateNoteRequest.newBuilder().setTitle("with a picture").build())).getNote().getId();
    }

    private String upload(String noteId, String contentType, byte[] image) {
        return ((UploadNoteImageResponse) rpc.procedure("UploadNoteImage").invoke(UploadNoteImageRequest.newBuilder()
                .setNoteId(noteId)
                .setContentType(contentType)
                .setImage(ByteString.copyFrom(image))
                .build())).getImageUrl();
    }

    @Test
    void anUploadedImageIsStoredAndPubliclyReadable() throws Exception {
        String noteId = createNote();

        String url = upload(noteId, "image/png", ImageUploadTest.PNG);

        assertThat(url).startsWith(MINIO.getS3URL() + "/notes-it/" + FAMILY + "/" + noteId + "/").endsWith(".png");
        HttpResponse<byte[]> fetched = HttpClient.newHttpClient().send(
                HttpRequest.newBuilder(URI.create(url)).GET().build(), HttpResponse.BodyHandlers.ofByteArray());
        assertThat(fetched.statusCode()).isEqualTo(200);
        assertThat(fetched.body()).isEqualTo(ImageUploadTest.PNG);
        assertThat(fetched.headers().firstValue("Content-Type")).hasValue("image/png");
        assertThat(upload(noteId, "image/png", ImageUploadTest.PNG)).isEqualTo(url);
    }

    @Test
    void anUploadIsRefusedForBadInputOrWithoutEditAccess() {
        String noteId = createNote();

        assertThat(catchThrowableOfType(ConnectException.class,
                () -> upload(noteId, "image/png", ImageUploadTest.JPEG)).code())
                .isEqualTo(ConnectCode.INVALID_ARGUMENT);
        assertThat(catchThrowableOfType(ConnectException.class,
                () -> upload(noteId, "application/pdf", ImageUploadTest.PNG)).code())
                .isEqualTo(ConnectCode.INVALID_ARGUMENT);

        rpc.procedure("ShareNote").invoke(ShareNoteRequest.newBuilder()
                .setNoteId(noteId)
                .setSubject(ShareSubject.SHARE_SUBJECT_MEMBER)
                .setMemberUserId(OTHER)
                .setPermission(SharePermission.SHARE_PERMISSION_VIEW)
                .build());
        actAs(OTHER);
        assertThat(catchThrowableOfType(ConnectException.class,
                () -> upload(noteId, "image/png", ImageUploadTest.PNG)).code())
                .isEqualTo(ConnectCode.PERMISSION_DENIED);

        actAs("55555555-5555-5555-5555-555555555555");
        assertThat(catchThrowableOfType(ConnectException.class,
                () -> upload(noteId, "image/png", ImageUploadTest.PNG)).code())
                .isEqualTo(ConnectCode.NOT_FOUND);
    }
}
