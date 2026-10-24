.. _user-webui-transfers:

####################
Suivi des transferts
####################

Section *Supervision*, adresse ``/webui/transfers``. La consultation requiert
le droit de lecture sur les transferts ; toute action (mettre en pause,
reprendre, annuler, relancer, ajouter, pré-enregistrer) requiert le droit
d'écriture sur les transferts, comme l'API REST.

Un transfert est l'échange d'un fichier selon une :ref:`règle <user-add-rule>`,
avec un partenaire distant (transfert *client*, à l'initiative de la Gateway)
ou pour un partenaire qui se connecte à un serveur local (transfert
*serveur*). Tant qu'il peut encore s'exécuter, il est dans la table des
transferts ; terminé ou annulé, il passe à l'historique. L'écran montre les
deux ensemble.


Liste
=====

En tête de page, une rangée compacte donne le nombre de transferts de chaque
statut : *Tous*, *En cours*, *En erreur*, *Planifiés*, *En pause*,
*Interrompus*, *Attendus*, *Terminés*, *Annulés*. Un clic sur un statut
n'affiche que ses transferts ; le statut choisi est souligné et en gras. Une
infobulle sur chaque statut dit ce qu'il signifie.

La barre d'outils recherche un transfert par le nom ou le chemin de son
fichier, tels que la colonne *Fichier* les affiche (:ref:`règle commune
<user-webui-search>`, majuscules et minuscules confondues, accents compris),
ou par son identifiant de suivi exact. Le bouton *Filtres* ouvre les
critères : règle, sens (envoi ou réception), partenaire ou serveur (une liste en
deux groupes, partenaires distants et serveurs locaux), compte (les comptes de
chaque partenaire ou serveur, ceux du seul partenaire choisi quand il l'est),
protocole (la famille, avec sa variante TLS), période de début. Un filtre
s'applique dès qu'il change ; le
nombre de filtres actifs s'affiche sur le bouton, *Réinitialiser* les
retire tous. Les nombres de la rangée des statuts suivent la recherche et les
filtres.

La liste montre un transfert par ligne et occupe toute la largeur de l'écran.
Un texte trop long pour sa colonne se termine par « … » ; il se lit en entier
dans l'infobulle de sa cellule, comme les informations secondaires.

.. list-table::
   :header-rows: 1
   :widths: 20 80

   * - Colonne
     - Contenu
   * - (sens)
     - Une icône : une flèche montante pour un envoi, descendante pour une
       réception, dans un nuage quand la Gateway est le serveur du transfert.
       L'infobulle le dit en toutes lettres (« Envoi depuis le serveur vers le
       client »).
   * - N°
     - Le numéro du transfert.
   * - Statut
     - Le statut du transfert (voir ci-dessous), en gras et dans sa couleur.
       Pour un transfert en erreur, l'infobulle donne la cause et le message.
   * - Fichier
     - Le fichier à sa destination, lien vers la fiche du transfert : pour un
       envoi, son chemin chez le partenaire ; pour une réception, son chemin
       sur la Gateway ; ses noms tant que le transfert n'a pas commencé. Un
       chemin trop long est coupé au début (« …/in/releve.csv ») : le nom du
       fichier reste visible. En infobulle, le chemin entier.
       La colonne prend toute la place que les autres laissent.
   * - Règle
     - La règle ; en infobulle, le partenaire, le compte et le protocole.
   * - Partenaire
     - Le partenaire distant, ou le serveur local d'un transfert serveur ; en
       infobulle, le compte et le protocole.
   * - Début
     - Le début ; pour un transfert planifié, l'heure prévue ; pour un
       transfert attendu, sa date limite.
   * - Fin
     - La fin ; pour un transfert en erreur qui sera retenté, l'heure de la
       prochaine tentative, précédée de l'icône de reprise.
   * - Progression
     - Une barre qui porte le pourcentage quand la taille du fichier est
       connue, les octets transférés sinon.
   * - Taille
     - La taille du fichier une fois connue (« 1,2 Mo ») ; en infobulle, la
       taille à l'octet près.

Les colonnes suivent l'ordre de l'écran historique. Les dates s'écrivent
partout de la même façon, à la seconde (« 21/10/2026 18:00:00 ») ; l'infobulle
les répète en toutes lettres. Quand la largeur utile est inférieure à
1 500 pixels environ, la colonne *Partenaire* se replie : le partenaire et le
compte se lisent dans l'infobulle de la règle. Sur un écran étroit, le tableau
défile dans son cadre.

Par défaut, les derniers transferts créés viennent en premier : le tri se
fait sur le numéro, décroissant. Les colonnes *N°*, *Fichier*, *Statut*,
*Règle*, *Début* et *Fin* se trient d'un clic
sur leur en-tête ; le statut se trie dans l'ordre de la rangée des statuts. Le
pied de liste indique les lignes affichées, permet de changer de page et de
choisir le nombre de lignes par page (10, 25, 50 ou 100 ; 25 par défaut). Il
est repris au-dessus du tableau, à droite de la recherche, pour changer de page
sans descendre. La recherche, les filtres, le statut, le tri et la page restent
dans l'adresse de la page et sont conservés pour la session, même quand ils
changent sans recharger la page : revenir à l'écran retrouve la liste telle
qu'elle a été laissée.

Le menu « ⋯ » de chaque ligne mène à la fiche du transfert et propose les
actions que son statut permet (voir *Actions*).


Chargement et actualisation
---------------------------

L'écran s'ouvre aussitôt : l'en-tête, la recherche et les filtres s'affichent,
la rangée des statuts et le tableau en attente. La rangée des statuts (avec le
nombre de transferts) et le tableau (avec ses pieds de liste) se lisent ensuite
en même temps, chacun s'affichant dès qu'il est prêt. Quand une lecture demandée
dure, un message *Chargement en cours…* s'affiche en haut à droite jusqu'à ce
qu'elle aboutisse.

Un changement de statut, de tri ou de page se charge sans recharger l'écran.
Après chaque affichage, la page suivante puis la précédente sont lues d'avance :
passer à l'une d'elles est immédiat. En revenant à la liste depuis la fiche d'un
transfert, la liste réapparaît aussitôt telle qu'elle a été laissée, à la même
page, puis se met à jour.

La rangée des statuts, le nombre de transferts et le tableau sont actualisés
toutes les 5 secondes, sans recharger la page, en gardant la recherche, les
filtres, le statut, le tri et la page. L'heure de la dernière lecture s'affiche
en haut de page (*Actualisé à …*) ; le bouton *Actualiser* lit sans attendre.
L'actualisation attend pendant une saisie dans la barre d'outils et tant
qu'un menu « ⋯ » est ouvert. Si une lecture échoue (session expirée,
Gateway injoignable), un message persistant donne l'heure de la dernière
lecture réussie ; l'écran réessaie au cycle suivant, sans s'arrêter.

Sans JavaScript, l'écran s'affiche en une fois, liste comprise, et ne
s'actualise pas.


Statuts
=======

Dans la liste, le libellé du statut s'écrit en gras dans sa couleur.

.. list-table::
   :header-rows: 1
   :widths: 18 18 64

   * - Statut
     - Couleur
     - Signification
   * - Planifié
     - Gris
     - Transfert client qui attend son heure de départ.
   * - Attendu
     - Gris
     - Transfert serveur pré-enregistré, qui attend que le partenaire se
       connecte, jusqu'à sa date limite.
   * - En cours
     - Vert
     - Le transfert s'exécute.
   * - En pause
     - Gris
     - Le transfert a été mis en pause par un utilisateur.
   * - Interrompu
     - Or
     - Le transfert a été interrompu de façon inattendue, par un arrêt de la
       Gateway par exemple.
   * - En erreur
     - Rouge
     - Le transfert a échoué. Il reste dans la table des transferts : il
       peut être retenté automatiquement, repris ou relancé.
   * - Terminé
     - Vert
     - Le transfert, ses traitements avant et après transfert compris, s'est
       terminé sans erreur.
   * - Annulé
     - Gris
     - Le transfert a été annulé par un utilisateur.


Fiche d'un transfert
====================

Le nom du fichier est le titre de la fiche ; le statut et le numéro du transfert
sont affichés dessous. La fiche réunit :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Carte
     - Contenu
   * - Erreur
     - Pour un transfert en erreur : la cause, libellé lisible du code
       d'erreur, avec le code technique (``TeConnection``…) ; le message tel
       que la Gateway l'a conservé ; l'étape où le transfert s'est arrêté.
   * - Transfert
     - La règle et le sens, le partenaire ou le serveur, le compte, le
       client, le protocole, l'identifiant distant et l'identifiant de suivi.
   * - Fichier
     - Les noms à la source et à la destination, le chemin sur la Gateway et
       le chemin chez le partenaire, la taille. Pour un transfert qui n'a pas
       encore démarré, les chemins prévus, calculés comme la Gateway le fera
       au départ.
   * - Déroulement
     - Le début et la fin, l'étape et la tâche en cours, la progression.
   * - Tentatives
     - Pour un transfert client en cours de vie : les tentatives restantes,
       la prochaine tentative, le délai et son facteur d'augmentation.
   * - Informations de transfert
     - Les informations attachées au transfert, clé et valeur. Les clés
       réservées de la Gateway (``__followID__``…) sont nommées par ce
       qu'elles sont, leur nom technique dessous.

Le code d'erreur de la Gateway regroupe des causes voisines : un fichier déjà
existant ou occupé chez un partenaire PeSIT donne par exemple *Accès refusé*.
Le message conserve souvent le détail donné par le partenaire.


Actions
=======

Les actions dépendent du statut du transfert ; le serveur vérifie le statut à
chaque action. Une action que le statut ne permet pas affiche une page qui dit
pourquoi.

.. list-table::
   :header-rows: 1
   :widths: 22 78

   * - Action
     - Effet et statuts concernés
   * - Mettre en pause
     - Suspend un transfert en cours, ou un transfert client planifié avant
       son départ.
   * - Reprendre
     - Reprend un transfert client en pause, interrompu ou en erreur, là où
       il s'est arrêté. Un transfert serveur reprend quand le partenaire se
       reconnecte ; le transfert enfant d'un transfert synchrone se reprend
       par son parent.
   * - Annuler le transfert
     - Après une page de confirmation qui nomme le transfert, son fichier et
       son statut : le transfert passe à l'historique, au statut *Annulé* ; un
       transfert en cours est d'abord interrompu.
   * - Relancer
     - Crée un nouveau transfert avec la même règle, le même partenaire, le
       même compte, le même client et le même fichier, maintenant ou à une
       date choisie ; le transfert d'origine reste tel quel et la fiche du
       nouveau s'affiche. Réservé aux transferts client en pause,
       interrompus, en erreur, terminés ou annulés. Si la règle, le
       partenaire, le compte ou le client n'existe plus, la relance est
       refusée.

Quand la Gateway fonctionne en grappe, un transfert en cours sur un autre
nœud ne se met pas en pause et ne s'annule pas depuis celui-ci : la page le
dit, l'action se fait depuis l'interface du nœud qui l'exécute.


Ajouter un transfert
====================

Le bouton *Ajouter un transfert* ouvre la page *Nouveau transfert* : un
transfert client, planifié à l'heure de départ choisie (maintenant si elle
est vide).

.. list-table::
   :header-rows: 1
   :widths: 28 72

   * - Champ
     - Contenu
   * - Fichier
     - À l'envoi, le chemin du fichier, relatif au dossier d'envoi de la
       règle ; à la réception, le nom du fichier à demander.
   * - Nom à la destination
     - Facultatif ; vide, le même nom que le fichier.
   * - Règle
     - Les règles, groupées par sens : la règle fixe le sens du transfert.
   * - Partenaire et compte
     - Les comptes distants, groupés par partenaire.
   * - Client
     - Facultatif ; vide, le client du protocole du partenaire. Quand
       plusieurs clients existent pour ce protocole, il faut en choisir un.
   * - Départ
     - Facultatif, à l'heure de la Gateway ; vide, maintenant.
   * - Informations de transfert
     - Facultatives : une ligne par information, avec son nom et sa
       valeur ; *Ajouter une information* ajoute une ligne, le bouton de
       chaque ligne la retire. Les traitements de la règle peuvent les lire.
   * - Tentatives en cas d'erreur
     - Facultatives : nombre de tentatives, premier délai en secondes,
       facteur d'augmentation ; vides, celles du client.


Pré-enregistrer un transfert
============================

Le menu « ⋯ » de l'en-tête propose *Pré-enregistrer un transfert* : un
transfert serveur, que le partenaire échangera en se connectant au serveur
local, jusqu'à une date limite. Le transfert apparaît au statut *Attendu*.
La page demande le fichier, la règle, le serveur et le compte local avec
lequel le partenaire se connectera, et la date limite ; les informations de
transfert sont facultatives.


Ancienne page
=============

L'ancienne page de suivi des transferts reste accessible à son adresse,
``/webui/transfer_monitoring``, jusqu'à la version suivante ; la barre
latérale mène au nouvel écran.
