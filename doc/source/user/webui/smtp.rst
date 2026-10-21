.. _user-webui-smtp:

#################
Identifiants SMTP
#################

Section *Traitements*, adresse ``/webui/smtp``. La consultation requiert le
droit de lecture sur les règles ; la création et la modification, le droit
d'écriture ; la suppression, le droit de suppression.

Un identifiant SMTP est le compte d'envoi du traitement
:doc:`/reference/tasks/email` : l'adresse de l'expéditeur, le serveur SMTP
(hôte et port) et, au besoin, l'identifiant et le mot de passe de connexion à
ce serveur. Un traitement ``EMAIL`` désigne l'identifiant par l'adresse de
l'expéditeur (paramètre ``sender``) ; il y ajoute ses destinataires et un
modèle d'e-mail. L'adresse de l'expéditeur est unique.

La connexion au serveur dépend du port :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Port
     - Connexion
   * - 465
     - TLS dès la connexion.
   * - Autres (587, 25…)
     - STARTTLS si le serveur le propose ; sinon, la connexion n'est pas
       chiffrée.

Le certificat du serveur est vérifié. Sans identifiant, la Gateway envoie les
e-mails sans s'authentifier.

L'interface ne réaffiche jamais le mot de passe : ni dans la liste, ni sur la
fiche, ni dans le formulaire.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Expéditeur
     - L'adresse de l'expéditeur ; elle ouvre la fiche de l'identifiant.
   * - Serveur SMTP
     - L'hôte et le port du serveur.
   * - Authentification
     - L'identifiant de connexion, ou *Aucune*.
   * - Utilisé par
     - Le nombre de règles dont les traitements ``EMAIL`` désignent
       l'identifiant.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les colonnes *Expéditeur* et *Serveur SMTP* se trient d'un clic sur leur
en-tête (par défaut, l'expéditeur croissant). La recherche filtre les
identifiants dont l'adresse commence par le texte saisi et propose les
adresses existantes. Le compteur, à droite, indique le nombre d'identifiants
et, quand une recherche est active, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, le tri et la page sont
portés par l'adresse de la page et conservés pour la durée de la session du
navigateur, comme pour les autres listes.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur l'identifiant, selon
les droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir la fiche
     - lecture
     - La fiche de l'identifiant (voir ci-dessous).
   * - Modifier l'identifiant
     - écriture
     - Le formulaire de l'identifiant.
   * - Supprimer l'identifiant
     - suppression
     - La page de confirmation de la suppression.

Sans identifiant, la liste le dit et propose d'en ajouter un. Sans droit
d'écriture, le bouton *Ajouter un identifiant* et les actions qui modifient un
identifiant n'apparaissent pas ; le serveur refuse de toute façon une action
sans le droit correspondant.


Fiche d'un identifiant
======================

La fiche d'un identifiant, adresse ``/webui/smtp/<id>``, montre en lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Expéditeur
     - L'adresse de l'expéditeur et son rôle.
   * - Serveur SMTP
     - L'hôte, le port, la connexion que le port implique, l'identifiant de
       connexion (ou *Aucune authentification*) et, s'il y en a un, si le mot
       de passe est défini. Un identifiant enregistré par l'API REST sans hôte
       ou avec le port 0 est signalé : les traitements ne pourraient pas se
       connecter.
   * - Utilisé par
     - Les règles dont les traitements ``EMAIL`` désignent l'identifiant,
       chacune avec son sens (envoi ou réception) ; le nom de la règle ouvre
       ses traitements, où l'expéditeur se change. Une règle qui porte le même
       nom dans les deux sens figure deux fois.

Le bouton *Modifier l'identifiant* et le menu « ⋯ » (supprimer) sont sur la
fiche ; *Retour à la liste* ramène à la liste telle qu'on l'a quittée.


Créer ou modifier un identifiant
================================

Le bouton *Ajouter un identifiant* ouvre le formulaire de création ;
*Modifier l'identifiant* ouvre le même formulaire. Il se compose de trois
blocs :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Expéditeur
     - L'adresse de l'expéditeur, seule, sans nom affiché (``alertes@exemple.fr``) ;
       unique.
   * - Serveur SMTP
     - L'hôte (nom ou adresse IP) et le port, de 1 à 65535 ; la création
       propose 587.
   * - Authentification
     - L'identifiant de connexion, facultatif, et le mot de passe, dans un
       champ masqué. À la modification, laisser le mot de passe vide le
       conserve ; vider l'identifiant retire l'authentification et efface le
       mot de passe. Un mot de passe sans identifiant est refusé.

Quand des règles emploient l'identifiant, son adresse ne se modifie pas : les
traitements la désignent. Le formulaire la montre en lecture ; le serveur et
l'authentification restent modifiables.

*Valider* enregistre l'identifiant avec une notification et ramène à la liste
(création) ou à la fiche de l'identifiant (modification). Un formulaire refusé
est réaffiché avec, sous chaque champ fautif, ce qu'il faut corriger ; le mot
de passe est à saisir de nouveau.


Supprimer un identifiant
========================

La suppression d'un identifiant qu'aucune règle n'emploie se confirme sur une
page qui rappelle l'expéditeur et le serveur.

Un identifiant que des règles emploient ne se supprime pas : leurs traitements
``EMAIL`` échoueraient au prochain transfert. La page le dit et liste ces
règles, dont le nom ouvre les traitements : il faut d'abord y changer
l'expéditeur.


Et par l'API
============

Les mêmes identifiants se gèrent par l':ref:`API REST <reference-rest-email>`
(``/api/email/credentials``) et par la :ref:`ligne de commande
<ref-cli-client-email>` (``waarp-gateway email credential``). Un identifiant
créé par l'un s'affiche et se modifie par les autres.

L'ancienne page *Identifiants SMTP*, qui renvoyait à la ligne de commande, mène
désormais à cet écran.
