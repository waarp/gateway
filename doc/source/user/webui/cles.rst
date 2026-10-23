.. _user-webui-keys:

#########################
Clés cryptographiques
#########################

Section *Traitements*, adresse ``/webui/keys``. La consultation requiert le
droit de lecture sur l'administration ; la création et la modification, le
droit d'écriture ; la suppression, le droit de suppression.

Les clés cryptographiques servent aux traitements des règles qui chiffrent,
déchiffrent, signent et vérifient les fichiers transférés. Une clé porte un
nom, unique, par lequel les traitements la désignent, un type et une valeur.

.. list-table::
   :header-rows: 1
   :widths: 20 40 40

   * - Type
     - Valeur
     - Traitements
   * - AES (``AES``)
     - La clé en base64 : 24 caractères pour une clé AES-128 (16 octets), 32
       pour une clé AES-192 (24 octets), 44 pour une clé AES-256 (32 octets).
       Secrète.
     - :doc:`/reference/tasks/encrypt` et :doc:`/reference/tasks/decrypt`,
       avec une méthode AES ; la même clé sert aux deux partenaires.
   * - HMAC (``HMAC``)
     - Le secret partagé avec le partenaire, en base64. Secrète.
     - :doc:`/reference/tasks/sign` et :doc:`/reference/tasks/verify`, avec
       une méthode HMAC.
   * - PGP publique (``PGP-PUBLIC``)
     - La clé publique du partenaire, exportée en armure ASCII.
     - Chiffrement des fichiers destinés au partenaire (``ENCRYPT``,
       ``ENCRYPT&SIGN``) et vérification de ses signatures (``VERIFY``,
       ``DECRYPT&VERIFY``).
   * - PGP privée (``PGP-PRIVATE``)
     - La clé privée de la Gateway, exportée en armure ASCII, sans phrase
       secrète. Secrète.
     - Déchiffrement des fichiers reçus (``DECRYPT``, ``DECRYPT&VERIFY``) et
       signature des fichiers envoyés (``SIGN``, ``ENCRYPT&SIGN``) ; elle
       chiffre et vérifie aussi, comme sa clé publique.

L'interface ne réaffiche jamais la valeur d'une clé secrète : ni dans la
liste, ni sur la fiche, ni dans le formulaire.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Clé
     - Le nom de la clé ; il ouvre sa fiche.
   * - Type
     - *AES*, *HMAC*, *PGP publique* ou *PGP privée*.
   * - Détail
     - La taille d'une clé AES (*AES-128*), *Secret défini* pour une clé
       HMAC, la première identité et l'identifiant d'une clé PGP ; *Illisible*
       quand la valeur enregistrée ne se lit pas.
   * - État
     - *Utilisable* ; pour une clé PGP, *Expire le …* dans les 30 jours qui
       précèdent son échéance, puis *Expirée le …*, ou *Révoquée* ;
       *Inutilisable*, avec la cause, pour une clé enregistrée que les
       traitements ne peuvent pas lire (voir ci-dessous). Chaque état a sa
       forme.
   * - Utilisée par
     - Le nombre de règles dont les traitements désignent la clé.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les colonnes *Clé* et *Type* se trient d'un clic sur leur en-tête (par
défaut, le nom croissant). La recherche filtre les clés dont le nom commence
par le texte saisi et propose les noms existants ; le bouton *Filtres* ouvre
les cases des types, appliquées dès qu'elles changent, et *Réinitialiser*
revient à la liste complète. Le compteur, à droite, indique le nombre de clés
et, quand une recherche ou un filtre est actif, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page et conservés pour la durée de la
session du navigateur, comme pour les autres listes.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur la clé, selon les
droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir la fiche
     - lecture
     - La fiche de la clé (voir ci-dessous).
   * - Modifier la clé
     - écriture
     - Le formulaire de la clé.
   * - Supprimer la clé
     - suppression
     - La page de confirmation de la suppression.

Sans clé, la liste le dit et propose d'en ajouter une. Sans droit
d'écriture, le bouton *Ajouter une clé* et les actions qui modifient une clé
n'apparaissent pas ; le serveur refuse de toute façon une action sans le
droit correspondant.

Une clé enregistrée par l'API REST ou la ligne de commande peut être
*Inutilisable* : une clé AES qui n'est pas en base64 ou dont la taille ne
convient pas, un secret HMAC qui n'est pas en base64, une clé PGP illisible, une clé publique enregistrée comme
privée ou l'inverse, une clé privée protégée par une phrase secrète (la
Gateway n'a pas de champ pour la phrase). Les traitements qui la désignent
échoueraient au prochain transfert ; il faut lui donner une nouvelle valeur
en modifiant la clé.


Fiche d'une clé
===============

La fiche d'une clé, adresse ``/webui/keys/<id>``, montre en lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Clé
     - Le type et ce qu'il permet.
   * - Valeur
     - L'état de la clé ; pour une clé secrète, *Définie, jamais affichée*,
       avec la taille d'une clé AES ; pour une clé PGP, ses identités, son
       empreinte, son identifiant, son algorithme, sa date de création et son
       échéance. *Afficher la clé publique* déplie la valeur d'une clé PGP
       publique, pour la copier.
   * - Utilisée par
     - Les règles dont les traitements désignent la clé, chacune avec son
       sens (envoi ou réception) ; le nom de la règle ouvre ses traitements,
       où la clé se change. Une règle qui porte le même nom dans les deux
       sens figure deux fois.

Le bouton *Modifier la clé* et le menu « ⋯ » (supprimer) sont sur la fiche ;
*Retour à la liste* ramène à la liste telle qu'on l'a quittée.


Créer ou modifier une clé
=========================

Le bouton *Ajouter une clé* ouvre le formulaire de création ; *Modifier la
clé* ouvre le même formulaire. Il se compose de deux blocs :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Clé
     - Le nom (unique) et le type.
   * - Valeur
     - Pour le type choisi : la clé AES ou le secret HMAC, dans un champ
       masqué ; pour une clé PGP, un fichier ``.asc`` à déposer ou l'armure à
       coller, le fichier l'emportant quand les deux sont donnés. À la
       modification, l'en-tête du bloc dit ce qui est en place : laisser la
       valeur vide la conserve, tant que le type ne change pas ; changer de
       type demande la valeur du nouveau type.

Le formulaire refuse ce que les traitements ne pourraient pas lire : une clé
AES qui n'est pas la base64 de 16, 24 ou 32 octets, un secret HMAC qui n'est
pas en base64, une clé PGP illisible, une
clé publique donnée comme privée ou l'inverse, et une clé privée protégée par
une phrase secrète, qu'il faut exporter sans phrase.

Quand des règles emploient la clé, son nom et son type ne se modifient pas :
les traitements les désignent. Le formulaire les montre en lecture et
n'offre que la valeur, pour la remplacer.

*Valider* enregistre la clé avec une notification et ramène à la liste
(création) ou à la fiche de la clé (modification). Un formulaire refusé est
réaffiché avec, sous chaque champ fautif, ce qu'il faut corriger ; le nom et
le type sont conservés, les valeurs secrètes sont à saisir de nouveau.


Supprimer une clé
=================

La suppression d'une clé qu'aucune règle n'emploie se confirme sur une page
qui rappelle la clé (type, détail).

Une clé que des règles emploient ne se supprime pas : leurs traitements
échoueraient au prochain transfert. La page le dit et liste ces règles, dont
le nom ouvre les traitements : il faut d'abord y changer la clé.


Et par l'API
============

Les mêmes clés se gèrent par l':ref:`API REST <rest_keys>` (``/api/keys``) et
par la :ref:`ligne de commande <ref-cli-client-keys>` (``waarp-gateway
key``). Une clé créée par l'un s'affiche et se modifie par les autres.

L'ancienne page *Clés cryptographiques*, qui renvoyait à la ligne de
commande, mène désormais à cet écran.
