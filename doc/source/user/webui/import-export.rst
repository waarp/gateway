.. _user-webui-config:

####################################
Import et export de la configuration
####################################

Section *Administration*, adresse ``/webui/config``. La page et l'export
requièrent le droit de lecture sur l'administration ; l'import, le droit
d'écriture. Ce sont les droits de l'API REST.

La Gateway sait exporter tout ou partie de sa configuration dans un fichier
JSON ou YAML, et importer un tel fichier : sauvegarder la configuration, la
reporter d'une instance à une autre (de la recette à la production), reprendre
un fichier préparé à la main. Le fichier a le format de référence de
:doc:`l'import et l'export </reference/backup>`, le même que celui des
commandes ``waarp-gatewayd export`` et ``import`` et de la ligne de commande
``waarp-gateway updateconf``. Le déploiement d'une configuration par Waarp
Manager passe par la tâche ``UPDATECONF`` et n'est pas concerné.

La page tient en deux cartes.


Exporter la configuration
=========================

Les catégories sont cochées, toutes par défaut : serveurs locaux (avec leurs
comptes et leur authentification), partenaires distants (de même), clients
locaux, règles de transfert, utilisateurs, instances cloud, SNMP, autorités
d'authentification, clés cryptographiques, e-mail (identifiants SMTP et
modèles), surveillances de partenaires. Le format est JSON ou YAML.
*Exporter la configuration* télécharge le fichier
``waarp-gateway-<instance>-<date>-<heure>.json`` (ou ``.yaml``).

**Le fichier contient les secrets en clair** : mots de passe des comptes, clés
privées, secrets des instances cloud, mots de passe SMTP. Seuls les mots de
passe des utilisateurs n'y sont qu'en hachage. Il se range comme un secret.


Importer une configuration
==========================

Le fichier (``.json``, ``.yaml`` ou ``.yml``, 64 Mo au plus) est le seul champ
obligatoire. Les catégories cochées sont les seules importées ; celles que le
fichier ne porte pas sont ignorées.

Deux options :

- *Redémarrer les services concernés* : les serveurs locaux, les clients et les
  surveillances touchés par l'import repartent sur leur nouvelle
  configuration ; sinon ils la prendront à leur prochain redémarrage.
- *Vider ces catégories avant l'import* : les objets actuels des catégories
  cochées sont supprimés avant l'import, et avec eux ce qui en dépend. La
  case *Je confirme la suppression* est alors obligatoire, et le bouton
  devient *Vider et importer*.

*Simuler l'import* joue l'import dans une transaction puis l'annule : il dit si
le fichier passerait, sans rien changer. *Importer* l'applique.

L'import est tout ou rien : la première erreur annule l'ensemble, et la page
l'affiche telle que le produit la formule (un nom de règle vide, une autorité
dont le certificat a expiré, un protocole inconnu). Les données importées
repassent par les contrôles de la Gateway : un fichier exporté d'une instance
dont un objet n'est plus valide peut être refusé à l'import.


Et par l'API
============

Le même export répond sur ``GET /api/updateconf`` (en-têtes ``Accept`` et
``Targets``) et le même import sur ``POST /api/updateconf`` (en-têtes
``Targets``, ``Dry-Run``, ``Reset`` et ``Restart``) ; voir
:doc:`la référence de l'API </reference/rest/updateconf/index>` et celle de
:doc:`la ligne de commande </reference/cli/client/updateconf/index>`.
