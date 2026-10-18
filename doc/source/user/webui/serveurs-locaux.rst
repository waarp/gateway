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
     - Le protocole du serveur.
   * - Adresse
     - L'adresse d'écoute, sous la forme ``hôte:port`` ; ``*:port`` quand le
       serveur écoute sur toutes les interfaces (hôte vide).
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les quatre colonnes se trient d'un clic sur leur en-tête (par défaut, le nom
croissant ; l'état se trie du plus sain au plus dégradé). La recherche filtre
les serveurs dont le nom commence par le texte saisi et propose les noms
existants ; le bouton *Filtres* ouvre les cases des protocoles, appliquées dès
qu'une case change, et *Réinitialiser* revient à la liste complète. Le
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
     - La page du serveur : état, adresse, démarrage automatique, nombre de
       comptes locaux, dossiers et options du protocole, en lecture.
   * - Démarrer le serveur / Arrêter le serveur
     - écriture
     - Démarre un serveur hors ligne ou en erreur, arrête un serveur en ligne.
       Le résultat s'affiche en notification ; si le serveur ne démarre pas,
       la notification donne la raison et reste affichée.
   * - Authentification
     - lecture
     - Les moyens d'authentification du serveur (certificats, clés).
   * - Comptes locaux
     - lecture
     - Les comptes avec lesquels les partenaires se connectent au serveur.
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
     - Le nom (unique sur la Gateway), le protocole, et la case *Ne pas
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
     - Les options du protocole choisi, qui apparaissent après le choix. Les
       listes d'algorithmes SSH et de suites de chiffrement TLS se composent
       en deux colonnes, les algorithmes disponibles à gauche et les
       algorithmes retenus à droite, dans l'ordre de préférence ; une liste
       vide accepte tous les algorithmes de la Gateway. Les suites TLS,
       nommées par leur code décimal puis leur nom (``49195 - TLS_ECDHE_…``,
       du code le plus élevé au plus bas), se choisissent jusqu'à TLS 1.2 ;
       en TLS 1.3, la bibliothèque les fixe.
       Le détail de chaque option se trouve dans la référence des
       :ref:`options de protocole <reference-proto-config>`.

*Valider* enregistre le serveur et ramène à la liste avec une notification ;
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


Et par l'API
============

Les mêmes opérations sont disponibles par l'API REST, sur les points
:http:get:`/api/servers`, :http:post:`/api/servers`,
:http:put:`/api/servers/(string:server_name)/start` et
:http:put:`/api/servers/(string:server_name)/stop` (voir
:ref:`reference-rest-api`), et en ligne de commande avec ``waarp-gateway
server``.
