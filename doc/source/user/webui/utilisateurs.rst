.. _user-webui-users:

############
Utilisateurs
############

Section *Administration*, adresse ``/webui/users``. La consultation requiert
le droit de lecture sur les utilisateurs ; la création et la modification, le
droit d'écriture ; la suppression, le droit de suppression.

Un utilisateur est un compte d'accès à l'administration de la Gateway : le
même identifiant et le même mot de passe servent à l'interface web, à l'API
REST et à la ligne de commande ``waarp-gateway user``. Il porte un
identifiant, unique sur la Gateway, un mot de passe, conservé haché et que
personne ne peut relire, et ses droits, par domaine.

À l'installation, la Gateway crée l'utilisateur ``admin`` (mot de passe
``admin_password``) avec tous les droits : changez ce mot de passe ou
remplacez ce compte dès que possible.


Les droits
==========

Les droits se donnent par domaine, en lecture, en écriture et en suppression.
Consulter un écran demande la lecture de son domaine, ajouter ou modifier un
objet l'écriture, supprimer un objet la suppression ; le serveur contrôle ces
droits sur chaque page et chaque action.

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Domaine
     - Ce qu'il couvre
   * - Transferts
     - Lancer, programmer, suivre, relancer ou annuler les transferts et
       consulter leur historique. Un transfert ne se supprime pas : ce
       domaine n'a pas de droit de suppression.
   * - Serveurs et clients locaux
     - Les :doc:`serveurs locaux <serveurs-locaux>` (configuration,
       démarrage, arrêt, comptes locaux, authentification) et les
       :doc:`clients locaux <clients-locaux>`, qui suivent les mêmes droits.
   * - Partenaires distants
     - Les :doc:`partenaires distants <partenaires-distants>`, leurs comptes
       distants et leur authentification.
   * - Règles de transfert
     - Les règles de transfert, leurs tâches et leurs accès, les
       identifiants SMTP et les modèles d'e-mail.
   * - Utilisateurs
     - Les utilisateurs de cet écran et leurs droits. Les utilisateurs qui
       détiennent l'écriture sur ce domaine sont les administrateurs : ce sont
       les seuls à pouvoir gérer les comptes.
   * - Administration
     - Les clés cryptographiques, les autorités d'authentification, les
       instances cloud, SNMP et les surcharges de configuration.

L'interface, l'API et la ligne de commande écrivent les droits d'un domaine
avec la même notation : ``r`` pour la lecture, ``w`` pour l'écriture, ``d``
pour la suppression, un tiret pour un droit absent (``rwd`` : tous les droits
du domaine, ``r--`` : la lecture seule, ``---`` : aucun droit). Toute
combinaison est possible.

Chaque utilisateur retrouve ses propres droits sur la page *Mon compte* du
menu du compte, où il change aussi son mot de passe.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Utilisateur
     - L'identifiant ; il ouvre la fiche. La mention *Vous* signale le compte
       connecté.
   * - Transferts, Serveurs, Partenaires, Règles, Utilisateurs,
       Administration
     - Les droits de l'utilisateur sur chaque domaine, dans la notation
       ci-dessus ; la légende sous le tableau la rappelle.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les sept colonnes se trient d'un clic sur leur en-tête (par défaut,
l'identifiant croissant ; une colonne de droits se trie du moins au plus de
droits). La recherche filtre les utilisateurs dont l'identifiant commence par
le texte saisi et propose les identifiants existants ; le bouton *Filtres*
ouvre deux listes, le *Domaine* et le *Droit*, appliquées dès qu'elles
changent : un domaine seul garde les utilisateurs qui y ont un droit, un droit
seul ceux qui le détiennent dans un domaine quelconque, *Aucun droit* ceux qui
n'en ont aucun dans le domaine choisi (ou nulle part). *Réinitialiser* revient
à la liste complète. Le compteur, à droite, indique le nombre d'utilisateurs
et, quand une recherche ou un filtre est actif, le nombre affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page et conservés pour la durée de la
session du navigateur, comme pour les autres listes.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur l'utilisateur,
selon les droits de l'utilisateur connecté :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir la fiche
     - lecture
     - La fiche de l'utilisateur : son identifiant et ses droits en toutes
       lettres (voir ci-dessous).
   * - Modifier l'utilisateur
     - écriture
     - Le formulaire de l'utilisateur, pré-rempli.
   * - Supprimer l'utilisateur
     - suppression
     - La page de confirmation de la suppression. L'action n'est pas proposée
       sur son propre compte.

Sans droit d'écriture, le bouton *Ajouter un utilisateur* et les actions qui
modifient un utilisateur n'apparaissent pas ; le serveur refuse de toute façon
une action sans le droit correspondant.


Fiche d'un utilisateur
======================

La fiche d'un utilisateur, adresse ``/webui/users/<id>``, montre en lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Utilisateur
     - L'identifiant, la mention *Vous* pour le compte connecté et, quand
       l'utilisateur est le seul à détenir l'écriture sur les utilisateurs, la
       mention *Dernier administrateur*.
   * - Droits
     - Un tableau, le même que sur la page *Mon compte* : pour chaque domaine,
       *Oui* ou *Non* en lecture, en écriture et en suppression.

Le bouton *Modifier l'utilisateur* et le menu « ⋯ » (supprimer) sont sur la
fiche ; *Retour à la liste* ramène à la liste telle qu'on l'a quittée.


Créer ou modifier un utilisateur
================================

Le bouton *Ajouter un utilisateur* ouvre le formulaire de création ;
*Modifier l'utilisateur* ouvre le même formulaire, pré-rempli. Il se compose
de deux blocs :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Identité
     - L'identifiant, unique sur la Gateway ; le mot de passe, saisi deux
       fois (72 caractères au plus). À la création, le mot de passe est
       obligatoire ; à la modification, le laisser vide conserve le mot de
       passe actuel. Le mot de passe n'est jamais réaffiché, même quand le
       formulaire est refusé.
   * - Droits
     - Un tableau de cases à cocher : pour chaque domaine, la lecture,
       l'écriture et la suppression (les transferts n'ont pas de droit de
       suppression). Une case décochée retire le droit.

Sur son propre compte, un encadré rappelle que les droits s'appliquent dès
l'enregistrement : un administrateur qui se retire le droit de lecture sur
les utilisateurs est ramené sur *Mon compte*, et le dernier administrateur
est prévenu que s'il se retire l'écriture sur les utilisateurs, plus personne
ne pourra gérer les comptes.

*Valider* enregistre l'utilisateur avec une notification et ramène à la liste
(création) ou à la fiche de l'utilisateur (modification). Un formulaire refusé
est réaffiché avec l'identifiant et les cases saisis et, sous chaque champ
fautif, ce qu'il faut corriger : identifiant vide ou déjà pris, mot de passe
manquant, saisies différentes, mot de passe trop long.


Supprimer un utilisateur
========================

La suppression se confirme sur une page qui nomme l'utilisateur, rappelle ses
droits et dit la conséquence : il ne pourra plus se connecter à l'interface,
à l'API ni à la ligne de commande, et sa session en cours sera fermée à sa
prochaine action.

Deux suppressions sont refusées, et la page le dit avant l'action :

* celle de son propre compte : demandez-la à un autre administrateur, comme
  avec l'API ;
* celle du dernier administrateur, le seul à détenir l'écriture sur les
  utilisateurs : donnez d'abord ce droit à un autre utilisateur.


Et par l'API
============

Les mêmes utilisateurs se gèrent par l':doc:`API REST
</reference/rest/users/index>` (``/api/users``) et par la :ref:`ligne de
commande <reference-cli-client-users>` (``waarp-gateway user``), avec la
même notation des droits et les opérateurs ``+``, ``-`` et ``=`` qui ajoutent,
retirent ou remplacent des droits. Un utilisateur créé par l'un s'affiche et
se modifie par les autres.
