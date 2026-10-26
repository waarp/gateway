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
(par défaut, le nom croissant). La recherche filtre les règles par leur nom ou
leur chemin (:ref:`règle commune <user-webui-search>`) ; le bouton *Filtres* ouvre les cases des
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
     - L'onglet *Tâches* de la règle (voir ci-dessous).
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

Les onglets *Tâches* et *Accès*, avec leur nombre, sont décrits ci-dessous.


Onglet Tâches
=============

L'onglet *Tâches* de la fiche, adresse ``/webui/rules/<id>/tasks``, montre les
traitements que la Gateway exécute pour chaque transfert de la règle. Sa
consultation requiert le droit de lecture sur les règles ; l'ajout, la
modification, le déplacement et la suppression d'une tâche, le droit
d'écriture, comme dans l'API REST.

**Trois chaînes.** Elles sont côte à côte, chacune une carte repliable avec son
nombre de tâches ; une carte repliée le reste au retour sur l'onglet :

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Chaîne
     - Exécutée
   * - Avant le transfert
     - Avant l'échange du fichier ; la première tâche en échec arrête la
       chaîne et met le transfert en erreur.
   * - Après le transfert
     - Après l'échange du fichier ; même arrêt à la première tâche en échec.
   * - En cas d'erreur
     - Quand le transfert est en erreur ; la première tâche en échec arrête la
       chaîne.

Les tâches s'y lisent dans leur ordre d'exécution, chacune une carte pliée :
son rang et son type (par exemple ``COPY``), dont le bouton ⓘ donne le libellé
et ce que fait la tâche. Le chevron déplie ses paramètres en clair. Une case
cochée se lit « Oui ». Un objet que la Gateway n'a plus (une clé, un
identifiant SMTP, un modèle d'e-mail supprimé depuis) est marqué
*Introuvable*, dès la carte pliée, avec un lien vers l'écran où le recréer. Le détail de chaque
type est dans :ref:`la référence des traitements <reference-tasks>`.

**Ajouter une tâche.** Chaque chaîne a son bouton *Ajouter une tâche*. La page
fait choisir le type dans une liste où l'on tape : chaque type y montre son
libellé en gras, son code et ce qu'il fait, groupé par famille (opérations de
fichiers, exécutables, transfert, altérations de fichiers, archivage, réseau,
cryptographie), et la saisie filtre sur les trois ; ses
paramètres s'affichent aussitôt, sur la même page, une carte repliable par
groupe (*Fichier*, *Échange*, *Exécution*…), côte à côte quand ils tiennent. Sous la
liste, le bouton ⓘ du type choisi dit ce que fait la tâche. Sans JavaScript, le
bouton *Continuer* affiche les paramètres du type choisi. La tâche s'ajoute à
la fin de la chaîne. Un paramètre se saisit selon sa nature :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Nature
     - Saisie
   * - Texte
     - Un chemin, une adresse, une liste de fichiers ; les
       :ref:`substitutions <reference-tasks-substitutions>` y sont admises.
   * - Case à cocher
     - Cochée, le paramètre vaut ``true`` ; décochée, il vaut ``false`` si la
       tâche l'avait déjà, sinon il est omis et la tâche prend sa valeur par
       défaut.
   * - Boutons joints
     - Deux ou trois valeurs possibles (le sens d'un pré-enregistrement, le
       sort d'un fichier refusé par ICAP, une méthode de signature) : un
       bouton par valeur, le bouton choisi marqué d'une coche. Une seule valeur
       possible est affichée, sans choix à faire.
   * - Nombre
     - Un entier, borné quand la tâche le borne (niveau de compression de 0 à
       9).
   * - Durée
     - Sous la forme ``30s``, ``5m`` ou ``1h30m`` ; en secondes pour les
       délais d'une tâche ``TRANSFER``. La durée maximale d'exécution d'une
       tâche ``EXEC`` se saisit ainsi et s'affiche en clair (« 1 min 30 s ») ;
       vide, elle est sans limite. Un programme qui la dépasse est interrompu
       et le transfert passe en erreur. Repliée, la tâche nomme son programme
       et sa limite.
   * - Liste
     - Les valeurs que la tâche accepte (méthodes de chiffrement, jeux de
       caractères) ou les objets de la Gateway (clés, identifiants SMTP,
       modèles d'e-mail, partenaires, comptes, clients, serveurs, règles).
   * - Informations de transfert
     - Celles d'une tâche ``TRANSFER`` ou ``PREREGISTER`` : une ligne par
       information (nom, valeur), comme sur la page qui ajoute un transfert,
       et la case qui copie d'abord celles du transfert en cours. Une valeur
       enregistrée et non modifiée garde son type (un nombre reste un
       nombre).
   * - Lignes clé=valeur
     - Les informations de transfert que pose une tâche ``SETINFO``, une par
       ligne ; une valeur vide retire la clé au moment de l'exécution.

Un paramètre qui ne compte que selon un autre ne s'affiche qu'avec lui : le
dossier où déplacer un fichier refusé par ICAP, la durée limite d'un transfert
synchrone.

**La tâche TRANSFER.** Sa page reprend celle qui ajoute un transfert : le
fichier et son nom à la destination ; le sens (envoi ou réception), qui ne
propose que les règles de ce sens ; le partenaire et son compte, en une seule
liste ; le client ; les informations de transfert ; les tentatives en cas
d'erreur, en secondes. Elle garde ce qui lui est propre : la copie des
informations du transfert en cours, et le mode, asynchrone (la tâche programme
le transfert et passe à la suite) ou synchrone (elle attend sa fin, avec une
durée limite facultative).

Le formulaire vérifie ce qu'il peut (paramètres obligatoires, nombres, JSON),
et un paramètre typé dès que l'on quitte son champ : des adresses e-mail (un
vrai nom d'hôte après « @ », plus strict que la Gateway, qui admet
``banque@banque?fr``), une durée, des droits en octal, l'adresse d'un serveur
ICAP. Une valeur enregistrée et laissée telle quelle n'est pas rejugée. La
Gateway contrôle ensuite la tâche (un jeu de caractères, une clé, un
partenaire qu'elle ne connaît pas) et son refus s'affiche sur la page, sans
rien enregistrer.

**Modifier une tâche.** Le menu « ⋯ » d'une tâche mène à sa page de
modification. Son type ne change pas : pour un autre type, supprimer la tâche
et en ajouter une. Les paramètres que l'écran ne connaît pas (posés par l'API
REST, par exemple) sont conservés tels quels, et la page le dit.

**Ordre et suppression.** Dans une chaîne de deux tâches ou plus, chaque tâche
a une poignée : on la glisse à sa nouvelle place ou, la poignée sélectionnée au
clavier, on la déplace avec les flèches haut et bas. Une tâche ne quitte pas sa
chaîne. Le nouvel ordre ne s'applique qu'avec *Valider l'ordre* ; *Annuler*
rétablit l'ordre affiché. *Monter* et *Descendre*, dans le menu « ⋯ » de la
tâche, l'échangent avec sa voisine sans la glisser. La page de suppression
rappelle la tâche et ses paramètres ; les tâches suivantes remontent d'un rang.
Chaque ajout, modification, déplacement, nouvel ordre ou suppression réécrit la
chaîne en une seule transaction, comme l'API REST.

**Chaîne modifiée entre-temps.** Une modification, un déplacement, un nouvel
ordre ou une suppression portent l'état de la chaîne telle que la page
l'affichait. Si elle a
changé depuis (un autre exploitant, un autre onglet du navigateur), rien n'est
enregistré : l'onglet revient avec l'état actuel de la chaîne et le dit. Un
ajout se fait toujours à la fin de la chaîne telle qu'elle est au moment de
l'enregistrement.


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
signale. L'état de la règle, puis les deux côtés, côte à côte, sont des cartes
repliables : une carte repliée le reste au retour sur l'onglet. Chaque accès y
est une carte, son type au-dessus de sa cible, avec son menu « ⋯ », qui mène à
sa page de retrait. Un accès ne porte que sa cible : il ne se modifie pas, on
retire l'un et on ajoute l'autre.

**Ajouter un accès.** Chaque côté a son bouton : *Ajouter un serveur ou un
compte local*, *Ajouter un partenaire ou un compte distant*. Il ouvre une page
où l'on tape ou choisit la cible, dans une liste groupée par type, parmi ceux de
ce côté : les serveurs et les partenaires avec leur protocole, les comptes avec
leur serveur ou leur partenaire. La saisie filtre la liste ; le nom d'un serveur
ou d'un partenaire trouve aussi ses comptes. Les cibles qui ont déjà accès ne
sont pas proposées. Sur une règle ouverte à tous, la page rappelle que ce
premier accès la ferme à tous les autres.

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
