# syntax=docker/dockerfile:1
FROM eclipse-temurin:26-jdk-alpine AS build

WORKDIR /src

COPY services/notes/gradlew services/notes/settings.gradle services/notes/build.gradle ./
COPY services/notes/gradle/ gradle/
RUN --mount=type=cache,target=/root/.gradle \
    ./gradlew --no-daemon dependencies --configuration runtimeClasspath

COPY services/notes/src/ src/
RUN --mount=type=cache,target=/root/.gradle \
    ./gradlew --no-daemon -x test bootJar && cp build/libs/*-SNAPSHOT.jar /app.jar

RUN java -Djarmode=tools -jar /app.jar extract --layers --launcher --destination /layers

FROM eclipse-temurin:26-jre-alpine

RUN apk add --no-cache curl \
 && addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=build --chown=app:app /layers/dependencies/ ./
COPY --from=build --chown=app:app /layers/spring-boot-loader/ ./
COPY --from=build --chown=app:app /layers/snapshot-dependencies/ ./
COPY --from=build --chown=app:app /layers/application/ ./

USER app:app
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:8080/actuator/health || exit 1

ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "org.springframework.boot.loader.launch.JarLauncher"]
