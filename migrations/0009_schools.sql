-- +goose Up
-- The schools a student can pick at sign-up and in settings, kept by the admins. users.school holds the code of the
-- school, as free text: removing a school here never touches the users who chose it.
CREATE TABLE schools (
  code text PRIMARY KEY CHECK (code ~ '^[A-Z0-9_-]{2,20}$'),
  name_fr text NOT NULL CHECK (char_length(name_fr) BETWEEN 1 AND 160),
  name_en text NOT NULL CHECK (char_length(name_en) BETWEEN 1 AND 160),
  name_ar text NOT NULL CHECK (char_length(name_ar) BETWEEN 1 AND 160),
  position integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO schools (code, name_fr, name_en, name_ar, position) VALUES
  ('ENIS', 'ENIS — École Nationale d’Ingénieurs de Sfax', 'ENIS — National Engineering School of Sfax', 'ENIS — المدرسة الوطنية للمهندسين بصفاقس', 1),
  ('ENETCOM', 'ENET’Com — École Nationale d’Électronique et des Télécommunications de Sfax', 'ENET’Com — National School of Electronics and Telecoms of Sfax', 'ENET’Com — المدرسة الوطنية للإلكترونيك والاتصالات بصفاقس', 2),
  ('ENIT', 'ENIT — École Nationale d’Ingénieurs de Tunis', 'ENIT — National Engineering School of Tunis', 'ENIT — المدرسة الوطنية للمهندسين بتونس', 3),
  ('INSAT', 'INSAT — Institut National des Sciences Appliquées et de Technologie', 'INSAT — National Institute of Applied Science and Technology', 'INSAT — المعهد الوطني للعلوم التطبيقية والتكنولوجيا', 4),
  ('ENSI', 'ENSI — École Nationale des Sciences de l’Informatique', 'ENSI — National School of Computer Science', 'ENSI — المدرسة الوطنية لعلوم الإعلامية', 5),
  ('SUPCOM', 'SUP’COM — École Supérieure des Communications de Tunis', 'SUP’COM — Higher School of Communications of Tunis', 'SUP’COM — المدرسة العليا للمواصلات بتونس', 6),
  ('ESPRIT', 'ESPRIT', 'ESPRIT', 'ESPRIT', 7),
  ('ENISO', 'ENISo — École Nationale d’Ingénieurs de Sousse', 'ENISo — National Engineering School of Sousse', 'ENISo — المدرسة الوطنية للمهندسين بسوسة', 8),
  ('ENIM', 'ENIM — École Nationale d’Ingénieurs de Monastir', 'ENIM — National Engineering School of Monastir', 'ENIM — المدرسة الوطنية للمهندسين بالمنستير', 9),
  ('ENIG', 'ENIG — École Nationale d’Ingénieurs de Gabès', 'ENIG — National Engineering School of Gabès', 'ENIG — المدرسة الوطنية للمهندسين بقابس', 10),
  ('ENICARTHAGE', 'ENICarthage — École Nationale d’Ingénieurs de Carthage', 'ENICarthage — National Engineering School of Carthage', 'ENICarthage — المدرسة الوطنية للمهندسين بقرطاج', 11),
  ('ISIMM', 'ISIMM — Institut Supérieur d’Informatique et de Mathématiques de Monastir', 'ISIMM — Higher Institute of Computer Science and Mathematics of Monastir', 'ISIMM — المعهد العالي للإعلامية والرياضيات بالمنستير', 12),
  ('ISI', 'ISI — Institut Supérieur d’Informatique', 'ISI — Higher Institute of Computer Science', 'ISI — المعهد العالي للإعلامية', 13),
  ('FST', 'FST — Faculté des Sciences de Tunis', 'FST — Faculty of Sciences of Tunis', 'FST — كلية العلوم بتونس', 14);

-- +goose Down
DROP TABLE schools;
