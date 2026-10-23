.. _user-webui-clouds:

###############
Instances cloud
###############

Section *Administration*, adresse ``/webui/clouds``. La consultation requiert
le droit de lecture sur l'administration ; la création et la modification,
le droit d'écriture ; la suppression, le droit de suppression.

Une instance cloud est un stockage distant que la Gateway emploie à la place
du disque local : une fois déclarée, son nom s'emploie dans les chemins des
règles, des serveurs et des transferts, sous la forme
``nom-instance:dossier/fichier`` (voir :ref:`les dossiers <instances-cloud>`).
Elle porte un nom, unique, un type parmi les cinq stockages que la Gateway
sait utiliser, une clé et un secret d'accès dont le sens dépend du type, et
des options propres au type. Le secret est conservé chiffré et n'est jamais
réaffiché, ni par l'interface ni par l'API.

À l'enregistrement, la Gateway instancie le stockage pour valider la
configuration : une configuration que le stockage refuse n'est pas
enregistrée, et la page dit la cause. Pour OneDrive, cette validation demande
un jeton à Microsoft : des identifiants faux sont refusés à l'enregistrement.


Les types de stockage
=====================

.. list-table::
   :header-rows: 1
   :widths: 22 30 48

   * - Type
     - Clé et secret
     - Options
   * - Amazon S3 (``s3``, ``aws``)
     - La clé d'accès et la clé secrète, ou les variables d'environnement
       ``AWS_ACCESS_KEY_ID`` et ``AWS_SECRET_ACCESS_KEY``.
     - *Bucket* (obligatoire), région, point d'accès (pour un service
       compatible S3), taille des morceaux envoyés, seuil de copie par
       morceaux.
   * - Azure Blob (``azblob``, ``azureblob``)
     - Le nom et la clé du compte de stockage, ou les variables
       ``AZURE_STORAGE_ACCOUNT``, ``AZURE_CLIENT_SECRET``, ``AZURE_TENANT_ID``
       et ``AZURE_CLIENT_ID``.
     - Point d'accès, taille des morceaux envoyés, objets lus par requête de
       listage. Le conteneur se nomme dans le chemin.
   * - Azure Files (``azfiles``, ``azurefiles``)
     - Comme Azure Blob.
     - *Partage* (obligatoire), point d'accès.
   * - Google Cloud Storage (``gcs``, ``google``)
     - Le chemin, sur la Gateway, du fichier d'identifiants du compte de
       service, ou le contenu JSON de ces identifiants en secret, ou la
       variable ``GOOGLE_APPLICATION_CREDENTIALS``.
     - *Bucket* et *numéro du projet* (obligatoires), politique d'accès
       uniforme.
   * - OneDrive (``onedrive``, ``sharepoint``)
     - L'identifiant et le secret de l'application enregistrée sur le portail
       Azure, avec l'autorisation d'application ``Files.ReadWrite.All`` ; la
       Gateway demande elle-même le jeton.
     - *Annuaire* (tenant), *identifiant du lecteur* et *type de lecteur*
       (obligatoires ; le formulaire propose *Professionnel*), région, taille
       des morceaux envoyés.

Le détail de chaque option, ses valeurs admises et sa valeur par défaut se
trouvent dans la :ref:`référence des instances cloud <reference-cloud>`. Les
autres réglages que la bibliothèque de stockage accepte se saisissent dans le
bloc *Autres options* du formulaire, une option par ligne sous la forme
``clé=valeur``. Le champ *Ajouter une option*, sous la zone, propose les options
que la bibliothèque accepte pour le type choisi, avec leur valeur par défaut et
leur description telles qu'elle les donne (en anglais) : la saisie filtre sur la
clé, puis sur la description ; Entrée ou un clic inscrit ``clé=`` sur une
nouvelle ligne de la zone, qui reste modifiable, et une option déjà inscrite
n'est plus proposée.


La liste
========

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Colonne
     - Contenu
   * - Instance cloud
     - Le nom de l'instance, tel qu'il s'écrit dans les chemins ; il ouvre sa
       fiche.
   * - Type
     - Le stockage, et entre parenthèses la clé enregistrée quand c'est un
       alias (« Amazon S3 (aws) »).
   * - Emplacement
     - Le bucket (S3, Google), le partage (Azure Files) ou l'identifiant du
       lecteur (OneDrive) ; rien pour Azure Blob, dont le conteneur se nomme
       dans le chemin.
   * - Authentification
     - *Clé renseignée*, *Variables d'environnement* ou *Aucune*.
   * - Actions
     - Le menu « ⋯ » de la ligne (voir ci-dessous).

Les quatre colonnes se trient d'un clic sur leur en-tête (par défaut, le nom
croissant). La recherche filtre les instances dont le nom commence par le
texte saisi et propose les noms existants ; le bouton *Filtres* ouvre les
cases des types de stockage, appliquées dès qu'elles changent, et
*Réinitialiser* revient à la liste complète. Le compteur, à droite, indique le
nombre d'instances et, quand une recherche ou un filtre est actif, le nombre
affiché.

Le pied de liste permet de changer de page et de choisir le nombre de lignes
par page (10 par défaut, 25, 50 ou 100). La recherche, les filtres, le tri et
la page sont portés par l'adresse de la page et conservés pour la durée de la
session du navigateur, comme pour les autres listes.

Le menu « ⋯ » de chaque ligne donne accès aux actions sur l'instance, selon
les droits de l'utilisateur :

.. list-table::
   :header-rows: 1
   :widths: 30 15 55

   * - Action
     - Droit
     - Effet
   * - Voir la fiche
     - lecture
     - La fiche de l'instance (voir ci-dessous).
   * - Modifier l'instance
     - écriture
     - Le formulaire de l'instance, pré-rempli.
   * - Supprimer l'instance
     - suppression
     - La page de confirmation de la suppression.

Sans instance, la liste propose directement d'en ajouter une. Sans droit
d'écriture, le bouton *Ajouter une instance cloud* et les actions qui
modifient une instance n'apparaissent pas ; le serveur refuse de toute façon
une action sans le droit correspondant.


Fiche d'une instance
====================

La fiche d'une instance, adresse ``/webui/clouds/<id>``, montre en lecture :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Contenu
   * - Instance
     - Le type et le chemin à employer pour la désigner
       (``nom-instance:dossier/fichier``).
   * - Accès
     - La clé, nommée selon le type (clé d'accès, nom du compte, fichier
       d'identifiants, identifiant d'application) ; si un secret est défini,
       sans jamais en montrer la valeur ; si les identifiants viennent des
       variables d'environnement.
   * - Options
     - Les options du type avec leur valeur ; quand l'option n'est pas
       renseignée, la valeur par défaut de la bibliothèque de stockage, que la
       Gateway applique, ou *Non défini* pour une option obligatoire ; puis les
       autres options, par leur clé. Une option que la bibliothèque de stockage
       traite comme sensible est nommée sans sa valeur (*Masquée*).
   * - Utilisée par
     - Les règles de transfert dont le dossier local nomme l'instance, les
       serveurs locaux dont la racine ou un dossier la nomme (chaque nom
       ouvre la :doc:`fiche du serveur <serveurs-locaux>`), le nombre de
       transferts dont le chemin local la nomme.

Le bouton *Modifier l'instance* et le menu « ⋯ » (supprimer) sont sur la
fiche ; *Retour à la liste* ramène à la liste telle qu'on l'a quittée.


Créer ou modifier une instance
==============================

Le bouton *Ajouter une instance cloud* ouvre le formulaire de création ;
*Modifier l'instance* ouvre le même formulaire, pré-rempli. Il se compose de
quatre blocs ; tous, sauf l'identité, se replient d'un clic sur leur titre, et
leur en-tête résume ce qu'ils contiennent (« À renseigner », « 2 options
renseignées », « Options par défaut de la passerelle »…) :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Bloc
     - Champs
   * - Identité
     - Le nom (unique ; ``file`` est réservé au disque local) et le type,
       parmi les cinq stockages. Changer de type vide les options du type
       précédent.
   * - Accès
     - La clé et le secret, dont le sens pour le type choisi est rappelé
       au-dessus des champs ; le secret est saisi masqué et jamais réaffiché :
       à la modification, le laisser vide conserve le secret actuel. Pour S3,
       Azure et Google, la case *Identifiants fournis par les variables
       d'environnement* dispense de la clé et du secret ; l'aide nomme les
       variables lues.
   * - Options du type
     - Les options du stockage choisi : les obligatoires sont marquées, l'aide
       dit la valeur que la Gateway applique quand l'option est vide (celle de
       la bibliothèque de stockage), les choix fermés (type de lecteur, région
       OneDrive) sont des listes. Un
       champ vide montre en grisé sa valeur par défaut ou un exemple de
       valeur (``archive``, ``s3.eu-west-3.amazonaws.com``…). Le bloc est
       ouvert quand il porte une option obligatoire, une valeur ou une
       erreur, replié sinon.
   * - Autres options
     - Une zone de texte, une option par ligne sous la forme ``clé=valeur``,
       pour les réglages de la bibliothèque de stockage que le formulaire ne
       nomme pas. Sous la zone, la liste des options que la bibliothèque
       accepte pour le type choisi : la clé, la valeur appliquée sans elle,
       sa description (en anglais) et, quand elles sont peu nombreuses, les
       valeurs proposées. Une ligne est refusée, avec sa raison, quand son
       option n'est pas connue du type (elle serait sans effet), quand c'est
       un identifiant ou une option sensible (elle serait conservée et
       affichée en clair : les identifiants passent par la clé et le secret),
       quand la Gateway l'impose elle-même, ou quand un champ du formulaire
       la porte déjà. Une option sensible enregistrée par l'API REST ou la
       ligne de commande n'est jamais affichée ; elle est conservée telle
       quelle tant que le type ne change pas.

*Valider* enregistre l'instance avec une notification et ramène à la liste
(création) ou à la fiche de l'instance (modification). Un formulaire refusé
est réaffiché avec les valeurs saisies (sauf le secret) et, sous chaque champ
fautif, ce qu'il faut corriger : nom vide, réservé ou déjà pris, type absent,
option obligatoire manquante, ligne d'option mal formée. Une configuration
que le stockage refuse revient avec la cause qu'il donne.

Une instance enregistrée sous un alias de type (``aws``, ``google``,
``sharepoint``…) garde cette clé tant que son type ne change pas.


Supprimer une instance
======================

La suppression se confirme sur une page qui nomme l'instance et liste les
règles, serveurs et transferts dont un chemin la nomme : après la
suppression, ces chemins ne fonctionneront plus. La Gateway ne refuse pas la
suppression ; c'est à l'exploitant de corriger ces chemins.


Et par l'API
============

Les mêmes instances se gèrent par l':doc:`API REST </reference/rest/clouds/index>`
(``/api/clouds``) et par la :ref:`ligne de commande
<reference-cli-client-cloud>` (``waarp-gateway cloud``), avec les mêmes types,
les mêmes options et le même secret jamais renvoyé. Une instance créée par
l'un s'affiche et se modifie par les autres.
