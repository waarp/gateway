.. _user-webui-partners:

####################
Partenaires distants
####################

Section *Connexions*, adresse ``/webui/partners``. La consultation requiert le
droit de lecture sur les partenaires ; la création et la modification, le
droit d'écriture ; la suppression, le droit de suppression.

Un partenaire distant est un serveur tiers auquel la Gateway se connecte pour
déposer ou retirer des fichiers. Il porte le protocole et l'adresse de ce
serveur, les options du protocole, ce que la Gateway vérifie de lui à la
connexion (son certificat, sa clé d'hôte ou son mot de passe) et ses *comptes
distants* : les identités que la Gateway lui présente, avec leurs moyens
d'authentification. Un transfert vers un partenaire nomme toujours un de ses
comptes et passe par un :doc:`client local <clients-locaux>` du même
protocole.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Partenaire distant
     - Le nom du partenaire ; il ouvre sa fiche.
   * - Protocole
     - La famille du protocole (R66, HTTP, SFTP, FTP, PeSIT, WebDAV, AS2) et,
       pour une variante sécurisée, un cadenas, dont l'info-bulle dit
       « Chiffré par TLS » : R66-TLS se lit « R66 » suivi du cadenas, HTTPS
       « HTTP » suivi du cadenas.
   * - Adresse
     - L'adresse ``hôte:port`` à laquelle la Gateway se connecte.
   * - Comptes
     - Le nombre de comptes distants du partenaire, ou *Aucun* : un
       partenaire sans compte ne peut recevoir aucun transfert.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les quatre colonnes se trient d'un clic sur leur en-tête (par défaut, le nom
croissant). La recherche filtre les partenaires par leur nom (:ref:`règle commune
<user-webui-search>`) et propose les noms existants ; le bouton *Filtres* ouvre les familles de protocole, en boutons joints dont plusieurs peuvent être
choisis, et le choix *Tous*, *Avec TLS* ou *Sans TLS*, sur une seule ligne
quand la largeur le permet, appliqués dès qu'ils changent, sans recharger la page (la liste se met
à jour en place et l'adresse suit), et *Réinitialiser* revient à la liste complète. Le compteur,
à droite, indique le nombre de partenaires et, quand une recherche ou un
filtre est actif, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page ; le tri et le nombre de lignes
par page sont conservés pour la durée de la session du navigateur, comme pour
les :doc:`serveurs locaux <serveurs-locaux>`.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur le partenaire,
selon les droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir le détail
     - lecture
     - La fiche du partenaire, onglet *Détails* (voir
       :ref:`user-webui-partner-sheet`).
   * - Authentification
     - lecture
     - L'onglet *Authentification* de la fiche : ce que la Gateway vérifie du
       partenaire à la connexion.
   * - Comptes distants
     - lecture
     - L'onglet *Comptes distants* de la fiche : les identités que la Gateway
       présente au partenaire.
   * - Voir les transferts
     - lecture des transferts
     - Le suivi des transferts, filtré sur le partenaire ; depuis un compte
       distant, filtré sur ce compte.
   * - Modifier le partenaire
     - écriture
     - Le formulaire du partenaire, pré-rempli.
   * - Supprimer le partenaire
     - suppression
     - La page de confirmation de la suppression.

Sans partenaire, la liste propose directement d'en ajouter un. Sans droit
d'écriture, le bouton *Ajouter un partenaire* et les actions qui modifient le
partenaire n'apparaissent pas ; le serveur refuse de toute façon une action
sans le droit correspondant.


Créer ou modifier un partenaire
===============================

Le bouton *Ajouter un partenaire* ouvre le formulaire de création ;
*Modifier le partenaire* ouvre le même formulaire, pré-rempli. Il se compose
de quatre blocs :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Identité
     - Le nom (unique sur la Gateway), le protocole, choisi par sa famille,
       et la case *Chiffrer la connexion par TLS* pour les familles qui
       existent avec et sans TLS (SFTP est chiffré par SSH).
   * - Adresse
     - L'hôte (nom ou adresse IP) et le port (de 1 à 65535) du serveur
       partenaire, tous deux obligatoires : c'est l'adresse à laquelle la
       Gateway se connecte.
   * - Options du protocole
     - Les options de la famille choisie, qui apparaissent après le choix.
       Pour PeSIT, l'identifiant que le partenaire attend, la reprise des
       transferts interrompus et les points de synchronisation se règlent
       *comme le client* (la valeur du client local s'applique), *activés*
       ou *désactivés* ; le délai d'attente de l'acquittement se saisit en
       heures, minutes et secondes. Les options d'une famille qui en a
       beaucoup sont groupées par thème (pour PeSIT : identification,
       reprise et synchronisation, messages, acquittement). Le détail de
       chaque option se trouve dans la référence des :ref:`options de
       protocole <reference-proto-config>`.
   * - Chiffrement TLS
     - Quand TLS est coché : la version minimale et les suites de
       chiffrement, composées en deux colonnes comme pour les serveurs ; une
       liste vide accepte toutes les suites de la Gateway.

Les blocs *Options du protocole* et *Chiffrement TLS* sont repliés tant
qu'ils sont à leur valeur par défaut ; leur en-tête résume ce qu'ils
contiennent, et un clic les ouvre. Un bloc qui porte une valeur ou une erreur
s'affiche toujours ouvert. Le bloc *Adresse* se replie de la même façon mais
s'ouvre toujours, l'adresse étant obligatoire ; son en-tête la rappelle.

*Valider* enregistre le partenaire avec une notification et ramène à la
liste (création) ou à la fiche du partenaire (modification). Un formulaire
refusé est réaffiché avec les valeurs saisies et,
sous chaque champ fautif, ce qu'il faut corriger. Changer le protocole d'un
partenaire existant rend caducs les moyens d'authentification et les comptes
qui ne correspondent plus au nouveau protocole : le formulaire le rappelle.


Supprimer un partenaire
=======================

La suppression se confirme sur une page qui nomme le partenaire et ses
conséquences : ses comptes distants et leurs moyens d'authentification sont
supprimés avec lui, ainsi que ses accès aux règles de transfert, et les
transferts vers ce partenaire ne pourront plus être créés.

Un partenaire dont un compte est utilisé par des transferts en cours ne peut
pas être supprimé : la page l'indique et renvoie vers le suivi des
transferts. Il en va de même quand une surveillance de dossiers nomme un de
ses comptes : il faut d'abord retirer cette surveillance, par l'API ou la
ligne de commande.


.. _user-webui-partner-sheet:

Fiche d'un partenaire
=====================

La fiche d'un partenaire, adresse ``/webui/partners/<id>``, regroupe en
trois onglets ce qui le concerne. Chaque onglet est une page à part entière,
avec sa propre adresse, sa recherche, son tri et sa pagination ; le fil
d'Ariane et le bouton *Retour à la liste* ramènent à la liste telle qu'on
l'a quittée.

.. list-table::
   :header-rows: 1
   :widths: 22 78

   * - Onglet
     - Contenu
   * - Détails
     - Le partenaire en lecture : protocole, adresse, nombre de comptes
       distants, le client utilisé, les règles de transfert autorisées et
       les options du protocole. Le bouton *Modifier le partenaire* et le
       menu « ⋯ » (supprimer) sont ici.
   * - Authentification
     - Ce que la Gateway vérifie du partenaire à la connexion : son
       certificat, sa clé d'hôte ou son mot de passe selon le protocole. Le
       compteur de l'onglet donne le nombre de moyens.
   * - Comptes distants
     - Les identités que la Gateway présente au partenaire, et ce que chacune
       présente pour s'authentifier.

Les deux onglets de liste s'ouvrent aussi depuis le menu « ⋯ » de la liste
des partenaires.

La carte *Client utilisé* de l'onglet *Détails* dit par quel client local
passent les transferts vers ce partenaire, selon la règle de la Gateway :

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Clients locaux du protocole
     - Ce que la carte indique
   * - Un seul
     - Les transferts passent par ce client ; son nom ouvre sa fiche.
   * - Plusieurs
     - Chaque transfert doit nommer son client ; la carte les liste.
   * - Aucun
     - La Gateway en créera un, nommé d'après le protocole, au premier
       transfert ; la carte propose d'en ajouter un (droit d'écriture sur les
       serveurs).

La carte *Règles de transfert autorisées* liste les règles réservées au
partenaire, c'est-à-dire celles dont un accès le nomme, lui ou l'un de ses
comptes distants (la carte dit lequel), avec leur sens ; chaque règle mène à
sa fiche. Les règles sans aucune restriction, que tout partenaire peut
utiliser, ne sont pas listées : une phrase les compte. Les accès aux règles
se gèrent depuis l'écran des règles de transfert.

Les cartes de la fiche se replient, sauf la première : le résumé de chaque
carte repliée reste visible (le client utilisé, le compte des règles).


Moyens d'authentification
-------------------------

Un *moyen d'authentification* est ce qu'une partie présente à l'autre pour
prouver son identité. Sur la fiche du partenaire, ce sont les moyens que la
Gateway *vérifie* du partenaire ; sur la fiche d'un compte distant, ceux que
la Gateway *présente* au partenaire avec ce compte. Les types proposés
dépendent du protocole du partenaire :

.. list-table::
   :header-rows: 1
   :widths: 22 39 39

   * - Protocole
     - Ce que la Gateway vérifie du partenaire
     - Moyens d'un compte distant
   * - R66
     - Mot de passe
     - Mot de passe
   * - R66-TLS
     - Mot de passe, certificat TLS de confiance
     - Mot de passe, certificat TLS et sa clé
   * - SFTP
     - Clé publique SSH (la clé d'hôte du partenaire)
     - Mot de passe, clé privée SSH
   * - PeSIT
     - Mot de passe
     - Mot de passe, pré-connexion PeSIT
   * - PeSIT-TLS
     - Mot de passe, certificat TLS de confiance
     - Mot de passe, certificat TLS et sa clé, pré-connexion PeSIT
   * - HTTPS, FTPS, WebDAV et AS2 sur HTTPS
     - Certificat TLS de confiance
     - Mot de passe, certificat TLS et sa clé
   * - HTTP, FTP, WebDAV, AS2
     - Aucun (la Gateway ne vérifie pas l'identité du partenaire)
     - Mot de passe

Un type qui ne peut exister qu'une fois (le mot de passe, la pré-connexion)
n'est plus proposé quand il est déjà présent. Le certificat R66 historique
s'ajoute à ces listes quand l'instance l'autorise (voir
:ref:`reference-auth-methods`).

Les tableaux, les pages de détail et les formulaires des moyens sont ceux de
la :ref:`fiche d'un serveur <user-webui-server-sheet>` : un contenu lisible
qui ne révèle jamais le secret, le dépôt ou le collage d'un fichier PEM, un
champ laissé vide à la modification qui conserve la valeur en place.

Un *certificat TLS de confiance* se garde seul : déposé avec ses
intermédiaires, seul le certificat du partenaire est conservé, et le
formulaire le dit. Pour faire confiance à l'autorité du partenaire plutôt
qu'à son certificat, déclarez-la dans :doc:`autorites`, racine seule, ou
racine et intermédiaires collés quand le partenaire ne présente que son
certificat. Le certificat qu'un *compte distant* présente au partenaire se
dépose, lui, avec sa chaîne, comme celui d'un :ref:`serveur local
<user-webui-server-sheet>`.

La *pré-connexion PeSIT* est ce que la Gateway envoie au partenaire avant la
connexion PeSIT, quand il l'attend : un identifiant et un mot de passe,
facultatifs l'un et l'autre, de huit caractères ASCII au plus chacun. La
fiche du moyen montre l'identifiant ; le mot de passe, conservé chiffré,
n'est jamais réaffiché. Elle ne sert que si l'option *Désactiver la
pré-connexion* du partenaire n'est pas cochée.


Comptes distants
----------------

Le tableau de l'onglet *Comptes distants* donne l'identifiant de chaque
compte et les types de ses moyens d'authentification ; un compte sans aucun
moyen est signalé, car la Gateway ne peut pas se connecter avec lui.
L'identifiant ouvre la fiche du compte, elle-même en deux onglets : *Détails*
(partenaire, identifiant, moyens) et *Authentification* (ses moyens, gérés
comme ceux du partenaire).

*Ajouter un compte* demande l'identifiant, unique sur le partenaire : c'est
celui que la Gateway présente. Le même formulaire permet de donner au compte
son premier moyen d'authentification, le plus souvent un mot de passe : le
compte et le moyen sont enregistrés ensemble, ou pas du tout. Les moyens
suivants s'ajoutent depuis la fiche du compte. *Modifier le compte* ne change
que son identifiant.

La suppression d'un compte supprime ses moyens avec lui ; les transferts
vers le partenaire qui le nomment ne pourront plus être créés. Elle est
refusée tant que des transferts en cours l'utilisent (la page renvoie vers
le suivi des transferts) ou qu'une surveillance de dossiers le nomme.


Et par l'API
============

Les mêmes opérations sont disponibles par l'API REST, sur les points
:http:get:`/api/partners`, :http:post:`/api/partners`, et, pour la fiche,
:http:get:`/api/partners/(string:partner)/certificates` et
:http:get:`/api/partners/(string:partner_name)/accounts` (voir
:ref:`reference-rest-api`), et en ligne de commande avec ``waarp-gateway
partner``.
