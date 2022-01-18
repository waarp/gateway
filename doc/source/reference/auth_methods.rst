.. _reference-auth-methods:

###########################
Méthodes d'authentification
###########################

========================
Authentification interne
========================

Le terme "authentification interne" correspond aux méthodes d'authentification
utilisée pour authentifier un agent externe sur la *gateway* dans l'optique
d'effectuer un transfert. Cela concerne à la fois les :term:`partenaires distants<partenaire>`,
lorsque la *gateway* agit comme client; et les :term:`comptes locaux<compte local>`,
lorsque la *gateway* agit comme serveur.

À l'heure actuelle, les formes d'authentification interne supportées dans la
*gateway* sont :

+--------------------+------------------------------+----------------------+---------------------------+----------------------------+
| Nom d'usage        | Nom du type                  | Protocoles supportés | Valeur primaire attendue  | Valeur secondaire attendue |
+====================+==============================+======================+===========================+============================+
| Mot de passe       | *password*                   | R66, R66-TLS, HTTP,  | Un mot de passe           | N/A                        |
|                    |                              | HTTPS & SFTP         |                           |                            |
+--------------------+------------------------------+----------------------+---------------------------+----------------------------+
| Certificat TLS de  | *trusted_tls_certificate*    | HTTPS & R66-TLS      | Un certificat TLS de      | N/A                        |
| Confiance          |                              |                      | confiance                 |                            |
+--------------------+------------------------------+----------------------+---------------------------+----------------------------+
| Clé publique SSH   | *ssh_public_key*             | SFTP                 | Une clé publique SSH      | N/A                        |
+--------------------+------------------------------+----------------------+---------------------------+----------------------------+

Autorité d'authentification
---------------------------

En plus des formes d'authentification spécifiées ci-dessus, la *gateway* intègre
également le concept d'*autorité d'authentification*. Une autorité d'authentification
représente un tiers de confiance auquel la *gateway* peut déléguer la vérification
de l'authentification. À l'heure actuelle, les types d'autorité suivants sont
supportés :

+-------------------------------+----------------------+----------------------+-----------------------------------+
| Nom d'usage                   | Nom du type          | Protocoles supportés | Valeur d'identité publique        |
+===============================+======================+======================+===================================+
| Autorité de certification TLS | *tls_authority*      | HTTPS & R66-TLS      | Le certificat TLS de l'autorité   |
+-------------------------------+----------------------+----------------------+-----------------------------------+
| Autorité de certification SSH | *ssh_cert_authority* | SFTP                 | La clé publique SSH de l'autorité |
+-------------------------------+----------------------+----------------------+-----------------------------------+

========================
Authentification externe
========================

Le terme "authentification externe" correspond aux méthodes d'authentification
utilisée par la *gateway* pour s'authentifier auprès d'un agent externe dans
l'optique d'effectuer un transfert. Cela concerne à la fois les :term:`serveur locaux<serveur>`,
lorsque la *gateway* agit comme serveur; et les :term:`comptes distants<compte distant>`,
lorsque la *gateway* agit comme client.

À l'heure actuelle, les formes d'authentification externe supportées dans la
*gateway* sont :

+----------------+-------------------+----------------------+--------------------------+----------------------------+
| Nom d'usage    | Nom du type       | Protocoles supportés | Valeur primaire attendue | Valeur secondaire attendue |
+================+===================+======================+==========================+============================+
| Mot de passe   | *password*        | R66, R66-TLS, HTTP,  | Un mot de passe          | N/A                        |
|                |                   | HTTPS & SFTP         |                          |                            |
+----------------+-------------------+----------------------+--------------------------+----------------------------+
| Certificat TLS | *tls_certificate* | HTTPS & R66-TLS      | Un certificat TLS        | Une clé privée             |
+----------------+-------------------+----------------------+--------------------------+----------------------------+
| Clé privée SSH | *ssh_private_key* | SFTP                 | Une clé privée SSH       | N/A                        |
+----------------+-------------------+----------------------+--------------------------+----------------------------+

=======================
Explications détaillées
=======================

Certificats TLS
---------------

Lors d'un transferts, il est possible (pour les protocoles le supportant) de
s'authentifier via l'échange de certificats TLS. Le parti souhaitant s'authentifier
(que ce soit un client ou un serveur) envoie son certificat à son partenaire, et
celui-ci vérifie que le certificat appartienne bien à l'agent souhaitant
s'authentifier.

Ainsi, pour qu'une *gateway* puisse s'authentifier via ce mécanisme, elle doit
donc posséder un certificat TLS à envoyer, ainsi que la clé privée associée à ce
certificat (pour pouvoir chiffrer les messages). Il s'agit donc de l'authentification
de type `tls_certificate`.

À l'inverse, pour qu'un tier puisse s'authentifier après de la *gateway* via cette
méthode, il faut que la *gateway* puisse vérifier le certificat qui lui est envoyé.
Il y a 3 cas de figure possible dans ce cas:

- Si le certificat est auto-signé, alors il doit être préalablement attaché à
  l'entité représentant le tiers (compte ou partenaire) pour être considéré
  "de confiance" (*trusted_tls_certificate*).
- Si le certificat a été signé par une autorité publique, connue du système
  d'exploitation, alors aucune action préalable n'est requise. Le certificat
  pourra être vérifié par la *gateway* normalement.
- Si le certificat a été signé par une autorité privée, alors cette autorité
  doit être renseignée au préalable avec son certificat. Une fois cela fait, tous
  les certificats tiers signés par cette autorité pourront être utilisés.

Clés SSH
--------

Le protocole SFTP étant basé sur SSH, il est possible d'utiliser des clés SSH
pour s'authentifier lors de transferts SFTP. Pour cela, le parti souhaitant
s'authentifier envoie sa clé publique à son partenaire.

Ainsi donc, pour qu'une *gateway* puisse s'authentifier de cette manière, elle
doit avoir une clé privé, et une clé publique (cette dernière est inclue dans la
clé privée). Il s'agit donc d'une valeur d'authentification de type `ssh_private_key`.

Réciproquement, pour qu'un tier puisse s'authentifier auprès de la *gateway*, cette
dernière dois préalablement connaître la clé publique de ce tier, pour pouvoir la
valider lorsque celui-ci la lui présente. Par conséquent, une valeur de type
`ssh_public_key` doit préalablement avoir été attachée au compte de ce tier.

Certificats SSH
---------------

Pour les protocoles basés sur SSH, la *gateway* supporte également l'authentification
via certificat SSH. Au lieu de présenter une clé publique, un tiers peut, à la
place, présenter un certificat SSH. Similairement au certificats TLS, ce
certificat doit avoir été signé par une autorité de confiance pour pouvoir être
utilisé. Par conséquent, l'autorité de certification doit préalablement avoir
été renseignée à la *gateway* pour pouvoir utiliser ces certificats.

L'avantage de cette méthode par rapport au clés publique SSH généralement utilisées
est qu'elle permet de réduire nettement la pré-configuration de la *gateway*, car
il n'y a plus besoin de renseigner la clé publique de chaque nouveau partenaire.
Il suffit de renseigner la clé publique de l'autorité de certification pour permettre
l'authentification de tous les partenaires ayant été certifiés par cette autorité,
et ce, même si leur clé publique change.