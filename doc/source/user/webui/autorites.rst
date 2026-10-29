.. _user-webui-authorities:

##############################
Autorités d'authentification
##############################

Section *Administration*, adresse ``/webui/authorities``. La consultation
requiert le droit de lecture sur l'administration ; la création et la
modification, le droit d'écriture ; la suppression, le droit de suppression.

Une autorité d'authentification est un tiers de confiance auquel la Gateway
délègue la vérification des certificats (voir :ref:`les autorités
d'authentification <reference-auth-methods>`). Un certificat présenté par un
partenaire ou par un compte est accepté s'il a été enregistré au préalable
dans la Gateway, s'il est signé par une autorité connue du système
d'exploitation, ou s'il est signé par l'une des autorités déclarées ici. Une
autorité porte un nom, unique, un type, son identité publique et, au besoin,
la liste des hôtes qu'elle est habilitée à certifier.

.. list-table::
   :header-rows: 1
   :widths: 30 25 45

   * - Type
     - Identité publique
     - Effet
   * - Autorité de certification TLS (``tls_authority``)
     - Le certificat de l'autorité, au format PEM : la racine seule, ou la
       racine et ses intermédiaires collés, pour les partenaires qui ne
       présentent que leur certificat.
     - Les certificats qu'elle signe sont acceptés pour les connexions TLS,
       sans être enregistrés au préalable.
   * - Autorité de certification SSH (``ssh_cert_authority``)
     - La clé publique de l'autorité, au format ``authorized_keys``.
     - Les certificats SSH qu'elle signe sont acceptés pour les connexions
       SFTP : ceux des utilisateurs qui se connectent à la passerelle, ceux
       des serveurs des partenaires.

Les protocoles concernés par chaque type sont donnés par la :ref:`référence
<reference-auth-methods>`.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Autorité
     - Le nom de l'autorité ; il ouvre sa fiche.
   * - Type
     - *TLS* ou *SSH*.
   * - Certificat ou clé
     - Le sujet du certificat (TLS) ou le type de la clé (SSH) ; *Illisible*
       quand l'identité enregistrée ne se lit pas.
   * - Validité
     - Pour une autorité TLS, l'état de son certificat : *Valide*, *Expire
       dans les 30 jours* ou *Expirée*, chacun avec sa forme ; rien pour une
       autorité SSH, dont la clé n'expire pas.
   * - Hôtes autorisés
     - *Tous les hôtes*, ou les deux premiers hôtes et le nombre des autres.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les colonnes *Autorité*, *Type*, *Validité* et *Hôtes autorisés* se trient
d'un clic sur leur en-tête (par défaut, le nom croissant) ; le tri par
validité place en tête le certificat qui expire le premier. La recherche
filtre les autorités par leur nom (:ref:`règle commune <user-webui-search>`) et propose les
noms existants ; le bouton *Filtres* ouvre les cases des types et des états
de validité, appliquées dès qu'elles changent, et *Réinitialiser* revient à
la liste complète. Le compteur, à droite, indique le nombre d'autorités et,
quand une recherche ou un filtre est actif, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page ; le tri et le nombre de lignes
par page sont conservés pour la durée de la session du navigateur, comme pour
les autres listes.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur l'autorité, selon
les droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir la fiche
     - lecture
     - La fiche de l'autorité (voir ci-dessous).
   * - Modifier l'autorité
     - écriture
     - Le formulaire de l'autorité, pré-rempli.
   * - Supprimer l'autorité
     - suppression
     - La page de confirmation de la suppression.

Sans autorité, la liste le dit : la Gateway ne se fie alors qu'aux autorités
du système et aux certificats enregistrés ; elle propose d'en ajouter une.
Sans droit d'écriture, le bouton *Ajouter une autorité* et les actions qui
modifient une autorité n'apparaissent pas ; le serveur refuse de toute façon
une action sans le droit correspondant.


Fiche d'une autorité
====================

La fiche d'une autorité, adresse ``/webui/authorities/<id>``, montre en
lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Autorité
     - Le type et ce qu'il permet.
   * - Identité publique
     - Pour un certificat : le sujet, l'émetteur, la période de validité avec
       son état et l'empreinte SHA-256 ; pour une clé : son type et son
       empreinte. *Afficher le certificat* (ou *la clé publique*) déplie la
       valeur enregistrée, pour la copier.
   * - Hôtes autorisés
     - Les hôtes que l'autorité est habilitée à certifier, ou *Tous les
       hôtes*, avec la règle de comparaison du type (voir ci-dessous).
   * - Certificats signés
     - Pour une autorité TLS, les certificats enregistrés dans la Gateway
       qu'elle signe, directement ou par un intermédiaire que la chaîne
       enregistrée contient : le nom du certificat, qui ouvre sa fiche, et son
       porteur (serveur local, partenaire ou compte). Les certificats SSH
       signés ne sont pas enregistrés dans la Gateway : le bloc n'existe pas
       pour une autorité SSH.

Le bouton *Modifier l'autorité* et le menu « ⋯ » (supprimer) sont sur la
fiche ; *Retour à la liste* ramène à la liste telle qu'on l'a quittée.


Les hôtes autorisés
===================

Une autorité sans hôte autorisé vaut pour tous les hôtes. Avec une liste, la
Gateway n'accepte les certificats qu'elle signe que pour ces hôtes :

- **Autorité TLS** : les hôtes sont comparés au nom d'hôte du serveur, sans
  port : celui de l'adresse du partenaire, ou celui du serveur local dont on
  enregistre le certificat. Une autorité restreinte ne vaut pas pour les
  certificats des comptes.
- **Autorité SSH** : les hôtes sont comparés à l'adresse du serveur SFTP du
  partenaire, avec son port (``sftp.exemple.fr:22``). Les certificats des
  utilisateurs qui se connectent à la passerelle sont acceptés quels que
  soient les hôtes.


Créer ou modifier une autorité
==============================

Le bouton *Ajouter une autorité* ouvre le formulaire de création ;
*Modifier l'autorité* ouvre le même formulaire, pré-rempli. Il se compose de
trois blocs ; les deux derniers se replient d'un clic sur leur titre, et leur
en-tête résume ce qu'ils contiennent (« À renseigner », le sujet du
certificat en place, « Tous les hôtes », « 2 hôtes ») :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Identité
     - Le nom (unique) et le type.
   * - Identité publique
     - Pour le type choisi, un fichier à déposer (``.pem``, ``.crt`` ou
       ``.cer`` pour un certificat, ``.pub`` pour une clé) ou le contenu à
       coller ; le fichier l'emporte quand les deux sont donnés. À la
       modification, le bloc dit ce qui est en place : laisser les champs
       vides conserve l'identité actuelle, tant que le type ne change pas ;
       changer de type demande l'identité publique du nouveau type.
   * - Hôtes autorisés
     - Un hôte par ligne, sous la forme que le type compare (voir
       ci-dessus) ; vide, l'autorité vaut pour tous les hôtes. Les lignes
       vides et les doublons sont ignorés.

*Valider* enregistre l'autorité avec une notification et ramène à la liste
(création) ou à la fiche de l'autorité (modification). Un formulaire refusé
est réaffiché avec les valeurs saisies et, sous chaque champ fautif, ce qu'il
faut corriger : nom vide ou déjà pris, type absent, identité publique absente
ou illisible pour le type choisi, certificat qui n'est pas celui d'une
autorité de certification, ou certificat qui ne se vérifie pas (un
certificat expiré, par exemple), avec la cause que donne la vérification.


Supprimer une autorité
======================

La suppression se confirme sur une page qui rappelle l'autorité (type,
identité publique, hôtes) et, pour une autorité TLS, liste les certificats
enregistrés qu'elle signe. Après la suppression, les certificats qu'elle
signe ne sont plus acceptés, sauf s'ils sont enregistrés dans la Gateway ou
signés par une autorité du système. La Gateway ne refuse pas la suppression ;
les certificats enregistrés restent en place.


Et par l'API
============

Les mêmes autorités se gèrent par l':doc:`API REST
</reference/rest/authorities/index>` (``/api/authorities``) et par la
:ref:`ligne de commande <reference-cli-client-authority>`
(``waarp-gateway authority``). Une autorité créée par l'un s'affiche et se
modifie par les autres.

L'ancienne page *Autorités d'authentification*, qui renvoyait à la ligne de
commande, mène désormais à cet écran.
