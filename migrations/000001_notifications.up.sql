create table if not exists Notifications (
  Id varchar(36) primary key not null default uuid(),
  Token varchar(36) not null,
  Destination varchar(36) not null,
  Created_At datetime default now(),
  Expires_At int default 60,
  Expired boolean default false
);