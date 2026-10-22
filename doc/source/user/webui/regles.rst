.. _user-webui-rules:

######################
Règles de transfert
######################

Section *Traitements*, adresse ``/webui/rules``. La consultation requiert le
droit de lecture sur les règles ; la création et la modification, le droit
d'écriture ; la suppression, le droit de suppression.

Une règle de transfert dit comment un fichier s'échange : dans quel sens,
entre quels dossiers, avec quels traitements, et par qui elle peut être
employée. Une règle se désigne par son nom et son sens : un même nom peut
servir une règle d'envoi et une règle de réception.

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Élément
     - Rôle
   * - Sens
     - *Envoi* : la Gateway envoie le fichier, depuis un client vers un
       partenaire, ou quand un partenaire vient le chercher sur un serveur.
       *Réception* : la Gateway reçoit le fichier, d'un partenaire qui le
       dépose sur un serveur, ou quand un client va le chercher.
   * - Chemin
     - Ce par quoi un serveur reconnaît la règle quand un partenaire se
       connecte (voir ci-dessous). Par défaut, le nom de la règle.
   * - Dossier local
     - Où le fichier est lu (envoi) ou déposé (réception). Vide : le dossier
       du serveur, sinon celui de la Gateway.
   * - Dossier distant
     - Où le fichier va chez le partenaire, pour un transfert lancé par la
       Gateway. Vide : le partenaire reçoit le fichier sous son seul nom.
   * - Dossier de réception temporaire
     - Où le fichier attend la fin de sa réception ; un dossier local. Vide :
       un dossier choisi par la Gateway, celui de la règle ou du serveur.
   * - Tâches
     - Les traitements exécutés avant le transfert, après, et en cas
       d'erreur (voir :ref:`reference-tasks`).
   * - Accès
     - Les serveurs, comptes locaux, partenaires et comptes distants
       autorisés à employer la règle. Sans accès, la règle est ouverte à
       tous ; dès qu'un accès est donné, elle est fermée à tous les autres.

**Comment un serveur reconnaît la règle.** En R66, en HTTP et en PeSIT non
standard, le partenaire nomme la règle. En SFTP, FTP, WebDAV, AS2 et PeSIT
standard, le serveur la déduit du chemin du fichier : il cherche une règle de
ce chemin, dans le sens attendu puis dans l'autre, puis remonte au dossier
parent. C'est pourquoi un chemin est unique dans un sens, et ne peut être ni
le parent ni l'enfant du chemin d'une autre règle de ce sens. Vers une
Gateway en SFTP ou en FTP, le dossier distant de la règle d'envoi doit donc
être le chemin de la règle de réception du partenaire.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Règle
     - Le nom de la règle ; il ouvre sa fiche.
   * - Sens
     - *Envoi* ou *Réception*, avec son icône.
   * - Chemin
     - Le chemin de la règle, coupé sur une ligne (la bulle le donne en
       entier).
   * - Tâches
     - Le nombre de tâches avant le transfert, après, et en cas d'erreur
       (« 1 · 2 · 0 »).
   * - Accès
     - *Ouverte à tous*, ou *Réservée* avec le nombre d'accès.
   * - Commentaire
     - Le commentaire de la règle, coupé sur une ligne.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les colonnes *Règle*, *Sens* et *Chemin* se trient d'un clic sur leur en-tête
(par défaut, le nom croissant). La recherche filtre les règles dont le nom ou
le chemin contient le texte saisi ; le bouton *Filtres* ouvre les cases des
sens, appliquées dès qu'elles changent, et *Réinitialiser* revient à la liste
complète. Le pied de liste permet de changer de page et de choisir le nombre
de lignes par page. La recherche, les filtres, le tri et la page sont portés
par l'adresse de la page et conservés pour la durée de la session du
navigateur, comme pour les autres listes.

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir la fiche
     - lecture
     - La fiche de la règle (voir ci-dessous).
   * - Modifier la règle
     - écriture
     - Le formulaire de la règle.
   * - Voir les tâches
     - lecture
     - La page des tâches de la règle.
   * - Voir les accès
     - lecture
     - L'onglet *Accès* de la règle (voir ci-dessous).
   * - Voir les transferts
     - lecture des transferts
     - Le :ref:`suivi des transferts <user-webui-transfers>`, filtré sur la
       règle et son sens.
   * - Supprimer la règle
     - suppression
     - La page de confirmation de la suppression.

Sans règle, la liste le dit et propose d'en ajouter une. Sans droit
d'écriture, le bouton *Ajouter une règle* et les actions qui modifient une
règle n'apparaissent pas ; le serveur refuse de toute façon une action sans le
droit correspondant.


Fiche d'une règle
=================

La fiche d'une règle, adresse ``/webui/rules/<id>``, a trois onglets.
L'onglet *Détails* montre en lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Règle
     - Le sens, le chemin et la façon dont les serveurs l'emploient, le
       commentaire, l'état des accès.
   * - Dossiers
     - Le dossier local, le dossier distant et le dossier de réception
       temporaire ; un dossier vide dit ce que la Gateway emploie à sa place.

L'onglet *Tâches*, avec leur nombre, mène à la page des tâches de la règle ;
l'onglet *Accès* est décrit ci-dessous.


Onglet Accès
============

L'onglet *Accès* de la fiche, adresse ``/webui/rules/<id>/access``, dit qui
peut employer la règle. Sa consultation requiert le droit de lecture sur les
règles ; l'ajout et le retrait d'un accès, le droit d'écriture.

**Ouverte à tous ou réservée.** Une règle sans accès est ouverte à tous : tout
serveur, partenaire ou compte peut l'employer. Dès qu'un accès est donné, elle
est fermée à tous les autres, des deux côtés :

.. list-table::
   :header-rows: 1
   :widths: 35 65

   * - Bloc
     - Contrôlé quand
   * - Quand un partenaire se connecte
     - Un partenaire se connecte à un serveur de la Gateway : le serveur ou le
       compte local doit avoir accès.
   * - Quand la passerelle se connecte
     - La Gateway se connecte à un partenaire : le partenaire ou le compte
       distant doit avoir accès.

Une règle réservée d'un seul côté est donc fermée de l'autre ; l'onglet le
signale. Chaque accès a son menu « ⋯ », qui mène à sa page de retrait.

**Ajouter un accès.** Le bouton *Ajouter un accès* ouvre une page où l'on
choisit le type (serveur, compte local, partenaire, compte distant) puis la
cible ; les comptes sont groupés par serveur ou par partenaire, et les cibles
qui ont déjà accès ne sont pas proposées. Sur une règle ouverte à tous, la page
rappelle que ce premier accès la ferme à tous les autres.

**Retirer un accès, rouvrir à tous.** La page de retrait d'un accès dit, pour
le dernier, que la règle redevient ouverte à tous. Le bouton *Rouvrir à tous*
retire tous les accès d'une règle réservée, après confirmation.


Ajouter ou modifier une règle
=============================

Le bouton *Ajouter une règle* ouvre la page *Nouvelle règle de transfert* ;
*Modifier la règle* ouvre le même formulaire, rempli. Une carte décrit la
règle, une autre ses dossiers.

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Champ
     - Règle de saisie
   * - Nom
     - Obligatoire, unique dans un sens, sans « / » (l'API ne pourrait plus
       désigner la règle), 100 caractères au plus.
   * - Sens
     - Obligatoire à la création. Il ne change plus ensuite : les tâches et
       les accès de la règle en dépendent ; pour l'autre sens, il faut créer
       une règle.
   * - Chemin
     - Facultatif ; vide, le nom de la règle. Unique dans un sens, ni parent
       ni enfant du chemin d'une autre règle de ce sens.
   * - Commentaire
     - Facultatif.
   * - Dossier local, dossier distant
     - Facultatifs : un chemin absolu, un chemin relatif au dossier du
       serveur ou de la Gateway, ou une instance cloud.
   * - Dossier de réception temporaire
     - Facultatif ; un dossier local, jamais une instance cloud.

Un formulaire refusé reste affiché avec les valeurs saisies, un message sous
chaque champ à corriger et un encadré en tête. Une règle enregistrée mène à sa
fiche, avec une notification. La modification d'une règle conserve ses tâches
et ses accès.

**Renommer une règle employée.** L'historique désigne chaque transfert passé
par le nom de sa règle. Quand le nom change et que l'historique emploie
l'ancien, la validation s'arrête une fois sur un encadré : après un renommage,
ces transferts ne pourront plus être relancés. Valider de nouveau renomme la
règle.


Supprimer une règle
===================

La page de confirmation dit ce qui part avec la règle, ses tâches et ses
accès, et ce qui reste : l'historique de ses transferts, sous son nom.

Une règle que des transferts ou des dossiers surveillés désignent encore ne
peut pas être supprimée. La page le dit, donne le nombre de ces transferts,
quel que soit leur état (en cours, programmés, en pause ou en erreur), avec
le lien vers eux dans le suivi des transferts, et le nom des dossiers
surveillés ; il faut annuler ou terminer ces transferts, changer ou supprimer
ces dossiers surveillés, puis revenir.
