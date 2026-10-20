.. _user-webui-servers:

################
Serveurs locaux
################

Section *Connexions*, adresse ``/webui/servers``. La consultation requiert le
droit de lecture sur les serveurs ; la création et la modification, le droit
d'écriture ; la suppression, le droit de suppression.

Un serveur local est un point d'écoute de la Gateway pour un protocole : les
partenaires s'y connectent pour déposer ou retirer des fichiers. Chaque
serveur porte une adresse d'écoute, des dossiers, les options de son
protocole, des comptes locaux et des moyens d'authentification (voir la
:ref:`gestion des serveurs <user-server-management>` en ligne de commande).


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Serveur local
     - Le nom du serveur.
   * - État
     - *En ligne* (rond vert), *Hors ligne* (anneau gris) ou *Erreur* (losange
       rouge), avec, en cas d'erreur, le message produit par le serveur (par
       exemple un port déjà utilisé). Les états sont ceux de l'écran
       :doc:`etat-des-services`.
   * - Protocole
     - La famille du protocole (R66, HTTP, SFTP, FTP, PeSIT, WebDAV, AS2) et,
       pour une variante sécurisée, le badge *TLS* : R66-TLS se lit « R66 »
       suivi de *TLS*, HTTPS « HTTP » suivi de *TLS*.
   * - Adresse
     - L'adresse d'écoute, sous la forme ``hôte:port`` ; ``*:port`` quand le
       serveur écoute sur toutes les interfaces (hôte vide).
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les quatre colonnes se trient d'un clic sur leur en-tête (par défaut, le nom
croissant ; l'état se trie du plus sain au plus dégradé). La recherche filtre
les serveurs dont le nom commence par le texte saisi et propose les noms
existants ; le bouton *Filtres* ouvre les cases des familles de protocole et
le choix *Avec ou sans TLS*, appliqués dès qu'ils changent, et *Réinitialiser*
revient à la liste complète. Le
compteur, à droite, indique le nombre de serveurs et, quand une recherche ou
un filtre est actif, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page : un lien vers la liste telle
qu'on la voit peut être partagé, et les pages d'action y ramènent telle qu'on
l'a quittée. Ils sont aussi conservés pour la durée de la session du
navigateur : revenir à la liste par la barre latérale la rouvre telle qu'on
l'a laissée ; *Réinitialiser* revient à la liste complète et oublie ces
critères.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur le serveur, selon
les droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir le détail
     - lecture
     - La fiche du serveur, onglet *Détails* : état, adresse, démarrage
       automatique, nombre de comptes locaux, dossiers et options du
       protocole, en lecture (voir :ref:`user-webui-server-sheet`).
   * - Démarrer le serveur / Arrêter le serveur
     - écriture
     - Démarre un serveur hors ligne ou en erreur, arrête un serveur en ligne.
       Le résultat s'affiche en notification ; si le serveur ne démarre pas,
       la notification donne la raison et reste affichée.
   * - Authentification
     - lecture
     - L'onglet *Authentification* de la fiche : l'identité que le serveur
       présente aux partenaires (certificat, clé d'hôte ou mot de passe).
   * - Comptes locaux
     - lecture
     - L'onglet *Comptes locaux* de la fiche : les comptes avec lesquels les
       partenaires se connectent au serveur.
   * - Modifier le serveur
     - écriture
     - Le formulaire du serveur, pré-rempli.
   * - Supprimer le serveur
     - suppression
     - La page de confirmation de la suppression.

Sans serveur, la liste propose directement d'en ajouter un. Sans droit
d'écriture, le bouton *Ajouter un serveur* et les actions qui modifient le
serveur n'apparaissent pas ; le serveur refuse de toute façon une action
sans le droit correspondant.


Créer ou modifier un serveur
============================

Le bouton *Ajouter un serveur* ouvre le formulaire de création ; *Modifier le
serveur* ouvre le même formulaire, pré-rempli. Il se compose de quatre blocs :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Identité
     - Le nom (unique sur la Gateway), le protocole, choisi par sa famille,
       la case *Chiffrer la connexion par TLS* pour les familles qui existent
       avec et sans TLS (SFTP est chiffré par SSH), et la case *Ne pas
       démarrer automatiquement au lancement de la passerelle*.
   * - Adresse d'écoute
     - L'hôte (nom ou adresse IP, facultatif : vide, le serveur écoute sur
       toutes les interfaces) et le port (de 1 à 65535). Deux serveurs ne
       peuvent pas écouter sur la même adresse.
   * - Dossiers
     - Le dossier racine, le dossier de réception, le dossier de réception
       temporaire et le dossier d'envoi, tous facultatifs. Un chemin relatif
       s'entend depuis le dossier racine de la Gateway ; quand un dossier
       racine est donné, les trois autres prennent par défaut ``in``, ``out``
       et ``tmp``.
   * - Options du protocole
     - Les options de la famille choisie, qui apparaissent après le choix,
       et, quand TLS est coché, le bloc *Chiffrement TLS* (version minimale,
       suites de chiffrement). Les
       listes d'algorithmes SSH et de suites de chiffrement TLS se composent
       en deux colonnes, les algorithmes disponibles à gauche et les
       algorithmes retenus à droite, dans l'ordre de préférence ; une liste
       vide accepte tous les algorithmes de la Gateway. Les suites TLS,
       nommées par leur code décimal puis leur nom (``49195 - TLS_ECDHE_…``,
       du code le plus élevé au plus bas), se choisissent jusqu'à TLS 1.2 ;
       en TLS 1.3, la bibliothèque les fixe.
       Le détail de chaque option se trouve dans la référence des
       :ref:`options de protocole <reference-proto-config>`.

Les blocs *Dossiers*, *Options du protocole* et *Chiffrement TLS* sont
repliés tant qu'ils sont à leur valeur par défaut ; leur en-tête résume ce
qu'ils contiennent, et un clic les ouvre. Un bloc qui porte une valeur ou une
erreur s'affiche toujours ouvert. Le bloc *Adresse d'écoute* se replie de la
même façon mais s'ouvre toujours, le port étant obligatoire ; son en-tête
rappelle l'adresse.

*Valider* enregistre le serveur avec une notification et ramène à la liste
(création) ou à la fiche du serveur (modification) ;
un serveur créé est hors ligne jusqu'à son démarrage (ou au prochain lancement
de la Gateway s'il démarre automatiquement). Un formulaire refusé est
réaffiché avec les valeurs saisies et, sous chaque champ fautif, ce qu'il
faut corriger. La modification d'un serveur en ligne prend effet à son
prochain démarrage.


Supprimer un serveur
====================

La suppression se confirme sur une page qui nomme le serveur et ses
conséquences : ses comptes locaux et leurs moyens d'authentification sont
supprimés avec lui, les règles qui le désignent dans leurs accès ne
s'appliquent plus à lui, et s'il est en ligne, il est arrêté.

Un serveur utilisé par des transferts en cours ne peut pas être supprimé : la
page l'indique et renvoie vers le suivi des transferts. Il faut annuler ces
transferts ou attendre leur fin.


.. _user-webui-server-sheet:

Fiche d'un serveur
==================

La fiche d'un serveur, adresse ``/webui/servers/<id>``, regroupe en trois
onglets ce qui le concerne. Chaque onglet est une page à part entière, avec
sa propre adresse, sa recherche, son tri et sa pagination ; le fil d'Ariane
et le bouton *Retour à la liste* ramènent à la liste telle qu'on l'a quittée.

.. list-table::
   :header-rows: 1
   :widths: 22 78

   * - Onglet
     - Contenu
   * - Détails
     - Le serveur en lecture : état, adresse d'écoute, démarrage automatique,
       nombre de comptes locaux, dossiers et options du protocole. Le bouton
       *Modifier le serveur* et le menu « ⋯ » (démarrer, arrêter, supprimer)
       sont ici.
   * - Authentification
     - L'identité que le serveur présente aux partenaires : un certificat TLS
       et sa clé pour les protocoles sécurisés par TLS, une clé d'hôte SSH
       pour SFTP, un mot de passe pour R66. Le compteur de l'onglet donne le
       nombre de moyens.
   * - Comptes locaux
     - Les comptes avec lesquels les partenaires se connectent, et ce que
       chacun présente pour s'authentifier.

Les deux onglets de liste s'ouvrent aussi depuis le menu « ⋯ » de la liste
des serveurs.


Moyens d'authentification
-------------------------

Un *moyen d'authentification* est ce qu'une partie présente à l'autre pour
prouver son identité. Sur la fiche du serveur, ce sont les moyens que le
serveur présente ; sur la fiche d'un compte local, ceux que le partenaire
présente avec ce compte. Les types proposés dépendent du protocole du
serveur :

.. list-table::
   :header-rows: 1
   :widths: 22 39 39

   * - Protocole
     - Identité du serveur
     - Moyens d'un compte local
   * - R66
     - Mot de passe
     - Mot de passe
   * - R66-TLS
     - Mot de passe, certificat TLS et sa clé
     - Mot de passe, certificat TLS de confiance
   * - SFTP
     - Clé privée SSH (clé d'hôte)
     - Mot de passe, clé publique SSH
   * - HTTPS, FTPS, PeSIT-TLS, WebDAV et AS2 sur HTTPS
     - Certificat TLS et sa clé
     - Mot de passe, certificat TLS de confiance
   * - HTTP, FTP, PeSIT, WebDAV, AS2
     - Aucun (le serveur n'authentifie pas les connexions)
     - Mot de passe

Un type qui ne peut exister qu'une fois (le mot de passe) n'est plus proposé
quand il est déjà présent. Le certificat R66 historique s'ajoute à ces listes
quand l'instance l'autorise (voir :ref:`reference-auth-methods`).

Le tableau d'un onglet *Authentification* donne le nom du moyen, son type et
un *contenu* lisible qui ne révèle jamais le secret : pour un certificat, le
sujet et la date d'expiration (*Valide jusqu'au…*, *Expire le…* quand elle
tombe sous trente jours, *Expiré le…* en rouge) ; pour une clé, son type et
son empreinte ; pour un mot de passe ou une clé privée, « défini, non
affiché ». La page d'un moyen détaille le sujet, l'émetteur, la validité et
l'empreinte SHA-256 d'un certificat, ou le type et l'empreinte d'une clé, et
montre la partie publique (certificat au format PEM, ligne de clé publique)
qui peut être transmise au partenaire. Un mot de passe ou une clé privée ne
sont jamais réaffichés, pas même à leur auteur.

*Ajouter un moyen d'authentification* ouvre un formulaire en deux blocs : le
nom et le type, puis le contenu du type choisi. Un fichier PEM (certificat,
clé) se dépose depuis le poste ou se colle dans le champ prévu à côté ; le
certificat TLS d'un serveur demande aussi sa clé privée. Un formulaire refusé
(contenu absent, PEM illisible, certificat qui ne correspond pas à la clé,
type déjà présent) est réaffiché avec le message sous le champ fautif. À la
modification, le type ne change pas et un champ laissé vide conserve la
valeur en place. La suppression d'un moyen requiert le droit de suppression
et se confirme sur une page.


Comptes locaux
--------------

Le tableau de l'onglet *Comptes locaux* donne l'identifiant de chaque compte
et les types de ses moyens d'authentification ; un compte sans aucun moyen
est signalé, car il ne peut pas se connecter. L'identifiant ouvre la fiche
du compte, elle-même en deux onglets : *Détails* (serveur, identifiant,
adresses IP autorisées, moyens) et *Authentification* (ses moyens, gérés
comme ceux du serveur).

*Ajouter un compte* demande l'identifiant, unique sur le serveur, et, à
volonté, les adresses IP depuis lesquelles le compte peut se connecter (une
par ligne, IPv4 ou IPv6, sans plage ; vide, toutes les adresses sont
acceptées). Le même formulaire permet de donner au compte son premier moyen
d'authentification, le plus souvent un mot de passe : le compte et le moyen
sont enregistrés ensemble, ou pas du tout. Les moyens suivants s'ajoutent
depuis la fiche du compte.

La suppression d'un compte supprime ses moyens avec lui ; elle est refusée
tant que des transferts en cours l'utilisent, et la page renvoie alors vers
le suivi des transferts.


Et par l'API
============

Les mêmes opérations sont disponibles par l'API REST, sur les points
:http:get:`/api/servers`, :http:post:`/api/servers`,
:http:put:`/api/servers/(string:server_name)/start` et
:http:put:`/api/servers/(string:server_name)/stop`, et, pour la fiche,
:http:get:`/api/servers/(string:server_name)/credentials` et
:http:get:`/api/servers/(string:server_name)/accounts` (voir
:ref:`reference-rest-api`), et en ligne de commande avec ``waarp-gateway
server``.
