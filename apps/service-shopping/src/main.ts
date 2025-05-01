import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { ValidationPipe } from '@nestjs/common';
import 'dotenv/config';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);

  app.enableCors();
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      transform: true,
      forbidNonWhitelisted: true,
    }),
  );
  app.setGlobalPrefix('shopping-service');

  await app.listen(process.env.PORT ?? 3001).then(() => {
    console.log(`Server is running on port ${process.env.PORT ?? 3001}`);
  });
}

void bootstrap();
