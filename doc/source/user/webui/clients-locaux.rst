.. _user-webui-clients:

###############
Clients locaux
###############

Section *Connexions*, adresse ``/webui/clients``. La consultation requiert le
droit de lecture sur les serveurs ; la création et la modification, le droit
d'écriture ; la suppression, le droit de suppression. Les clients locaux
suivent les droits des serveurs : la Gateway n'a pas de droit propre aux
clients.

Un client local est le côté sortant de la Gateway pour un protocole : c'est
lui qui se connecte aux partenaires distants pour déposer ou retirer des
fichiers. Il n'a ni comptes ni moyens d'authentification propres (ce sont les
comptes distants des partenaires qui les portent). Il porte une adresse locale
de sortie facultative, les réglages de reprise automatique des transferts et
les options de son protocole (voir la :ref:`référence de l'API
<reference-rest-api>` et la ligne de commande ``waarp-gateway client``).

Un transfert sortant passe par un client du protocole du partenaire : celui
que la demande de transfert nomme, sinon le client par défaut de ce protocole.
Un client sert tous les partenaires de son protocole ; plusieurs clients d'un
même protocole peuvent coexister (deux adresses de sortie, deux politiques de
reprise).


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Client local
     - Le nom du client ; il ouvre sa fiche.
   * - État
     - *En ligne* (rond vert), *Hors ligne* (anneau gris) ou *Erreur* (losange
       rouge), avec, en cas d'erreur, le message produit par le client. Les
       états sont ceux de l'écran :doc:`etat-des-services`.
   * - Protocole
     - La famille du protocole (R66, HTTP, SFTP, FTP, PeSIT, WebDAV, AS2) et,
       pour une variante sécurisée, un cadenas, dont l'info-bulle dit
       « Chiffré par TLS » : R66-TLS se lit « R66 » suivi du cadenas, HTTPS
       « HTTP » suivi du cadenas.
   * - Adresse locale
     - L'adresse ``hôte:port`` depuis laquelle le client sort, ou
       *Automatique* quand le système choisit l'interface et le port.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les quatre colonnes se trient d'un clic sur leur en-tête (par défaut, le nom
croissant ; l'état se trie du plus sain au plus dégradé). La recherche filtre
les clients dont le nom commence par le texte saisi et propose les noms
existants ; le bouton *Filtres* ouvre les cases des familles de protocole et
le choix *Avec ou sans TLS*, appliqués dès qu'ils changent, et *Réinitialiser*
revient à la liste complète. Le compteur, à droite, indique le nombre de
clients et, quand une recherche ou un filtre est actif, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page et conservés pour la durée de la
session du navigateur, comme pour les :doc:`serveurs locaux <serveurs-locaux>`.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur le client, selon
les droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir le détail
     - lecture
     - La fiche du client : état, adresse locale, démarrage automatique,
       reprise des transferts, options du protocole et partenaires joignables
       (voir ci-dessous).
   * - Démarrer le client / Arrêter le client
     - écriture
     - Démarre un client hors ligne ou en erreur, arrête un client en ligne.
       **Arrêter un client interrompt ses transferts en cours.** Le résultat
       s'affiche en notification ; si le client ne démarre pas, la
       notification donne la raison et reste affichée.
   * - Modifier le client
     - écriture
     - Le formulaire du client, pré-rempli.
   * - Supprimer le client
     - suppression
     - La page de confirmation de la suppression.

Sans client, la liste propose directement d'en ajouter un. Sans droit
d'écriture, le bouton *Ajouter un client* et les actions qui modifient le
client n'apparaissent pas ; le serveur refuse de toute façon une action sans
le droit correspondant.


Fiche d'un client
=================

La fiche d'un client, adresse ``/webui/clients/<id>``, montre en lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Client
     - L'état, le protocole (la famille et, le cas échéant, « chiffré par
       TLS » avec le nom du produit), l'adresse locale, le démarrage
       automatique.
   * - Reprise des transferts
     - Le nombre de tentatives automatiques après un échec, le délai avant la
       première reprise et le facteur de progression du délai ; ou *Aucune
       reprise automatique*.
   * - Options du protocole
     - Les options du protocole du client, avec la valeur par défaut de la
       Gateway quand l'option n'est pas renseignée.
   * - Partenaires joignables par ce client
     - Les partenaires distants de son protocole : un transfert vers l'un
       d'eux passe par ce client, sauf à en nommer un autre. Chaque nom
       ouvre la :doc:`fiche du partenaire <partenaires-distants>`.

Le bouton *Modifier le client* et le menu « ⋯ » (démarrer, arrêter,
supprimer) sont sur la fiche ; *Retour à la liste* ramène à la liste telle
qu'on l'a quittée.


Créer ou modifier un client
===========================

Le bouton *Ajouter un client* ouvre le formulaire de création ; *Modifier le
client* ouvre le même formulaire, pré-rempli. Il se compose de quatre blocs :

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
   * - Adresse locale
     - L'hôte et le port de sortie, tous deux facultatifs : vides, le système
       choisit l'interface et le port.
   * - Reprise des transferts
     - Le nombre de tentatives automatiques après un échec (0 : aucune
       reprise ; 127 au plus), le délai avant la première reprise, en heures,
       minutes et secondes (vide, la Gateway attend une seconde), et le
       facteur de progression du délai (1 : délai constant ; 2 : délai
       doublé à chaque reprise). Sous ces trois réglages, une phrase dit
       combien de temps les reprises prennent réellement (« 5 tentatives :
       la dernière 15 min 30 s après le premier échec, avec des attentes de
       30 s à 8 min »), une frise place chaque tentative au prorata du temps
       écoulé depuis le premier échec, et le dépliant *Détail des tentatives*
       donne, pour chacune, l'attente et le moment ; le tout se recalcule à
       chaque saisie. La durée des tentatives elles-mêmes s'ajoute à ces
       attentes. La fiche du client porte la même phrase.
   * - Options du protocole
     - Les options de la famille choisie et, quand TLS est coché, le bloc
       *Chiffrement TLS* (version minimale, suites de chiffrement). Les
       listes d'algorithmes se composent en étiquettes ordonnées, comme pour
       les serveurs ; une liste vide accepte tous les algorithmes de la
       Gateway.
       Le détail de chaque option se trouve dans la référence des
       :ref:`options de protocole <reference-proto-config>`.

Les blocs *Adresse locale*, *Reprise des transferts*, *Options du protocole*
et *Chiffrement TLS* sont repliés tant qu'ils sont à leur valeur par défaut ;
leur en-tête résume ce qu'ils contiennent, et un clic les ouvre. Un bloc qui
porte une valeur ou une erreur s'affiche toujours ouvert.

*Valider* enregistre le client avec une notification et ramène à la liste
(création) ou à la fiche du client (modification) ;
un client créé est hors ligne jusqu'à son démarrage (ou au prochain lancement
de la Gateway s'il démarre automatiquement). Un formulaire refusé est
réaffiché avec les valeurs saisies et, sous chaque champ fautif, ce qu'il
faut corriger.


Supprimer un client
===================

La suppression se confirme sur une page qui nomme le client et ses
conséquences : les transferts à venir ou interrompus qui le désignent ne
pourront plus s'exécuter, l'historique des transferts garde son nom, et s'il
est en ligne, il est arrêté et ses transferts en cours interrompus.

Un client utilisé par des surveillances de dossiers ne peut pas être
supprimé : la page l'indique. Il faut d'abord supprimer ces surveillances,
par l'API ou la ligne de commande.


Et par l'API
============

Les mêmes opérations sont disponibles par l'API REST, sur les points
:http:get:`/api/clients`, :http:post:`/api/clients`,
:http:put:`/api/clients/(string:client_name)/start` et
:http:put:`/api/clients/(string:client_name)/stop` (voir
:ref:`reference-rest-api`), et en ligne de commande avec ``waarp-gateway
client``.
