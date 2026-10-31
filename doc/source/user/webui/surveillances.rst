.. _user-webui-watchers:

###########################
Surveillances de partenaires
###########################

Section *Traitements*, adresse ``/webui/watchers``. La consultation requiert le
droit de lecture sur les serveurs ; la création, la modification, le démarrage,
l'arrêt et le lancement d'un passage, le droit d'écriture ; la suppression, le
droit de suppression. Ce sont les droits de l'API REST pour les mêmes objets.

Une surveillance de partenaire interroge, à intervalle régulier, un dossier
chez un partenaire distant et crée un transfert de réception pour chaque
fichier trouvé dont le nom correspond à son motif. Elle réunit :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Élément
     - Rôle
   * - Compte distant
     - Le compte avec lequel la passerelle se connecte chez le partenaire. Seuls
       les partenaires dont le protocole sait lister un dossier sont proposés :
       SFTP, FTP et FTPS, R66 et R66-TLS, WebDAV.
   * - Client local
     - Le client de la passerelle qui porte la connexion, du même protocole que
       le partenaire.
   * - Règle de réception
     - La règle sous laquelle chaque fichier trouvé est reçu ; son dossier
       distant est le dossier interrogé.
   * - Motif
     - Les fichiers retenus, par leur nom, avec ``*`` et ``?`` comme jokers et
       ``[ ]`` pour un jeu de caractères ; vide, tous les fichiers.
   * - Intervalle
     - Le temps entre deux interrogations, en minutes ou en heures, une minute
       au moins.

Chaque surveillance est un service de la passerelle, démarré avec elle sauf si
elle est enregistrée arrêtée. Les fichiers déjà transférés sont ignorés d'un
passage à l'autre, sauf si la surveillance en demande la reprise.

La liste
========

La liste présente une ligne par surveillance : son nom, le partenaire et le
compte, le client, la règle, le motif, l'intervalle et l'état de son service.

.. list-table::
   :header-rows: 1
   :widths: 20 80

   * - État
     - Signification
   * - En ligne
     - Le service tourne ; le dossier est interrogé à l'intervalle prévu.
   * - Arrêtée
     - Le service est arrêté : enregistré arrêté, arrêté depuis la liste, ou
       créé par un autre nœud et pas encore démarré sur celui-ci.
   * - Erreur
     - Le service a échoué ; la raison s'affiche au survol de l'état.

La recherche porte sur le nom (voir :ref:`user-webui-search`). Les filtres
retiennent les partenaires et les états choisis ; ils s'appliquent dès qu'ils
changent. Les colonnes *Surveillance*, *Partenaire* et *Règle* se trient. Le
nombre de lignes par page se choisit dans le pied de la liste ; l'état de la
liste (recherche, filtres, tri, page) suit l'utilisateur dans les pages
d'action et se conserve pour la session du navigateur.

Le nom de la surveillance mène à sa fiche. Le menu « ⋯ » de chaque ligne
donne accès aux actions, selon les droits :

* *Voir la fiche* ;
* *Démarrer* ou *Arrêter*, selon l'état du service ;
* *Lancer un passage* : le dossier est interrogé tout de suite, une fois, que
  le service tourne ou non ; la notification dit combien de transferts le
  passage a créés, ou pourquoi il a échoué (partenaire injoignable, dossier
  illisible) ;
* *Modifier* ;
* *Supprimer*.

Fiche d'une surveillance
========================

La fiche rappelle la surveillance (état du service et, en erreur, la raison ;
partenaire et compte ; client ; règle, qui mène à sa propre fiche ; motif ;
intervalle ; reprise des fichiers déjà transférés ; démarrage avec la
passerelle) et liste, dans une carte pliée par défaut dont l'en-tête compte
les fichiers, les derniers fichiers reçus :
les vingt derniers transferts de réception de la règle pour ce partenaire et
ce compte, avec leur date, leur heure et leur état, chacun menant à la fiche
du transfert. Le bouton *Voir tous les transferts* ouvre le suivi des
transferts filtré sur la surveillance.

Le bouton *Modifier* ouvre le formulaire ; le menu « ⋯ » porte le démarrage,
l'arrêt, le passage et la suppression.

Créer ou modifier une surveillance
==================================

Le bouton *Ajouter une surveillance* ouvre une page en trois cartes, les
deux dernières repliables : la surveillance elle-même (nom, motif, intervalle,
reprise des fichiers déjà transférés, démarrage avec la passerelle), le
partenaire et la règle de réception. Dans la carte du partenaire, le protocole
se choisit d'abord, parmi ceux qui savent lister un dossier ; le compte
distant, proposé par partenaire de ce protocole, et le client local de ce
protocole suivent. Les partenaires PeSIT, HTTP et AS2 ne s'y trouvent pas :
leur protocole ne liste pas de dossier.

Le nom est unique. L'intervalle se saisit en nombre entier de minutes ou
d'heures. Le client doit être du même protocole que le partenaire du compte.
Un formulaire refusé réaffiche chaque valeur saisie avec, sous le champ en
cause, ce qu'il faut corriger.

Une surveillance créée démarre aussitôt, sauf si la case *Démarrer avec la
passerelle* est décochée : elle est alors enregistrée arrêtée et se démarre
depuis la liste. Une surveillance en ligne que l'on modifie redémarre sur sa
nouvelle configuration dès l'enregistrement, comme par l'API REST.

Lorsqu'aucun partenaire ne peut être surveillé (aucun compte distant chez un
partenaire dont le protocole liste les dossiers), la page le dit et mène aux
partenaires distants.

Supprimer une surveillance
==========================

La page de confirmation rappelle la surveillance (nom, partenaire et compte,
règle, motif, intervalle) et sa conséquence : le service s'arrête, les
transferts déjà créés restent, plus aucun ne sera créé pour ce dossier.

Et par l'API
============

Les mêmes surveillances se gèrent par l':ref:`API REST
<reference-rest-filewatcher>` (``/api/filewatchers``) et par la
:doc:`ligne de commande </reference/cli/client/filewatcher/index>`
(``waarp-gateway filewatcher``). Une surveillance créée par l'un s'affiche et
se modifie par les autres.

L'état du service ne vit que dans le processus de la passerelle : il n'est
exposé ni par l'API REST ni dans la configuration exportée. Une surveillance
créée par l'API sans être démarrée apparaît *Arrêtée* jusqu'à son démarrage.
